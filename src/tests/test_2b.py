import random

from .helpers import *


def test_basic_agree():
    states = get_nodes_state(HOSTS)
    leader = get_leader(states)

    for index in range(1, 4):
        log = submit_log_entry(leader, 'some%d' % index * 10)
        assert log['index'] == index
        assert log['term']
        assert log['command'] == 'some%d' % index * 10

        time.sleep(1)

        nodes = check_log_submitted(HOSTS - {leader}, log['index'], log['command'], log['term'])
        assert len(nodes) == len(HOSTS) - 1


def test_fail_agree():
    states = get_nodes_state(HOSTS)
    leader = get_leader(states)

    log = submit_log_entry(leader, 'ololo')
    assert len(check_log_submitted(HOSTS - {leader}, log['index'], log['command'], log['term'])) == len(HOSTS) - 1

    follower = random.choice(list(HOSTS - {leader}))
    docker_stop(follower)

    log = submit_log_entry(leader, 'alalala')
    assert len(check_log_submitted(HOSTS - {leader, follower},
                                   log['index'], log['command'], log['term'])) == len(HOSTS) - 2

    time.sleep(2)

    log = submit_log_entry(leader, 'ulululu')
    assert len(check_log_submitted(HOSTS - {leader, follower},
                                   log['index'], log['command'], log['term'])) == len(HOSTS) - 2
    docker_start(follower)
    time.sleep(1)

    states = get_nodes_state(HOSTS - {follower})
    leader = get_leader(states)
    assert all(s['commitIndex'] == log['index'] for s in states.values())

    submit_log_entry(leader, 'final')
    assert get_common_log_ids(HOSTS) == {1, 2, 3, 4}
    assert check_logs_equal(HOSTS)


def test_fail_no_agree():
    states = get_nodes_state(HOSTS)
    leader = get_leader(states)

    log = submit_log_entry(leader, 'ololo')

    time.sleep(1)

    states = get_nodes_state(HOSTS)
    assert all(s['commitIndex'] == log['index'] for s in states.values()), states

    peers = list(HOSTS - {leader})
    partition(peers[0], HOSTS)
    partition(peers[1], HOSTS)
    partition(peers[2], HOSTS)

    submit_log_entry(leader, 'alala')
    submit_log_entry(leader, 'ululu')

    time.sleep(1)

    # no majority - commit index should be the same
    states = get_nodes_state({leader})
    assert all(s['commitIndex'] == log['index'] for s in states.values()), states

    docker_restart(leader)
    docker_restart(peers[3])

    heal_partition(peers[0], HOSTS)
    heal_partition(peers[1], HOSTS)
    heal_partition(peers[2], HOSTS)

    time.sleep(2)

    assert get_common_log_ids(HOSTS) == {1}
    assert check_logs_equal(HOSTS)


def test_partition_leader():
    states = get_nodes_state(HOSTS)
    leader = get_leader(states)

    peers = list(HOSTS - {leader})
    partition(peers[0], {leader})
    partition(peers[1], {leader})
    partition(peers[2], {leader})

    time.sleep(2)
    states = get_nodes_state(HOSTS)
    assert get_leader(states) != leader
    assert check_one_leader(states, get_leader(states)), states


def test_rejoin():
    states = get_nodes_state(HOSTS)
    leader = get_leader(states)

    submit_log_entry(leader, 'ololo')
    time.sleep(1)

    # leader network failure
    partition(leader, HOSTS)
    wont_success = submit_log_entry(leader, 'wont success')  # index = 2

    time.sleep(1)

    states = get_nodes_state(HOSTS - {leader})
    second_leader = get_leader(states)
    new_log = submit_log_entry(second_leader, 'ohohoho')  # also index=2, but next term

    assert new_log['term'] > wont_success['term']

    # new leader network failure
    partition(second_leader, HOSTS)

    # heal old leader but it can't become new leader bc doesnt have all logs
    heal_partition(leader, HOSTS)

    time.sleep(1)

    states = get_nodes_state(HOSTS - {second_leader})

    new_leader = get_leader(states)

    assert check_one_leader(states, new_leader)
    assert new_leader != leader
    assert check_log_submitted(HOSTS, 2, 'ohohoho', new_log['term'])
    assert not check_log_submitted({leader}, 2, 'wont success', wont_success['term'])

    last_log = submit_log_entry(new_leader, 'last one')

    heal_partition(second_leader, HOSTS)
    time.sleep(1)

    states = get_nodes_state(HOSTS)

    assert all(s['commitIndex'] == last_log['index'] for s in states.values()), states


def test_backup():
    # put leader and one follower in a partition
    states = get_nodes_state(HOSTS)
    leader = get_leader(states)
    follower = list(HOSTS - {leader})[0]

    partition(leader, HOSTS - {leader, follower})
    partition(follower, HOSTS - {leader, follower})

    # submit lots of commands that won't commit
    for i in range(7):
        submit_log_entry(leader, 'first%d' % i)

    # in another partition should be new leader
    states = get_nodes_state(HOSTS - {leader, follower})
    partition_leader = get_leader(states)

    # lots of successful commands to new group.
    for i in range(3):
        l = submit_log_entry(partition_leader, 'second%d' % i)
        time.sleep(1)
        states = get_nodes_state(HOSTS - {leader, follower})
        assert all(s['commitIndex'] == l['index'] for s in states.values()), states

    # now another partitioned leader and one follower
    follower2 = list(HOSTS - {leader, follower})[0]
    partition(follower2, HOSTS)

    # lots more commands that won't commit
    for i in range(5):
        submit_log_entry(partition_leader, 'third %d' % i)

    # heal them all
    heal_partition(leader, HOSTS - {leader, follower})
    heal_partition(follower, HOSTS - {leader, follower})
    heal_partition(follower2, HOSTS)

    time.sleep(3)
    
    states = get_nodes_state(HOSTS)
    the_leader = get_leader(states)
    assert check_one_leader(states, the_leader)

    submit_log_entry(the_leader, 'last')

    time.sleep(1)

    assert check_logs_equal(HOSTS)
    assert len(get_common_log_ids(HOSTS)) in (4, 9)

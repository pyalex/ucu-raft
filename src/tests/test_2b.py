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

    states = get_nodes_state(HOSTS - {follower})
    assert all(s['commitIndex'] == log['index'] for s in states.values())

    docker_start(follower)

    log = submit_log_entry(leader, 'final')
    assert len(check_log_submitted(HOSTS - {leader},
                                   log['index'], log['command'], log['term'])) == len(HOSTS) - 1


def test_fail_no_agree():
    states = get_nodes_state(HOSTS)
    leader = get_leader(states)

    log = submit_log_entry(leader, 'ololo')

    time.sleep(1)

    states = get_nodes_state(HOSTS)
    assert all(s['commitIndex'] == log['index'] for s in states.values()), states

    peers = list(HOSTS - {leader})
    docker_stop(peers[0])
    docker_stop(peers[1])
    docker_stop(peers[2])

    submit_log_entry(leader, 'alala')
    submit_log_entry(leader, 'ululu')

    time.sleep(1)

    # no majority - commit index should be the same
    states = get_nodes_state({leader})
    assert all(s['commitIndex'] == log['index'] for s in states.values()), states

    docker_start(peers[0])
    docker_start(peers[1])
    docker_start(peers[2])

    new_log = submit_log_entry(leader, 'ohoho')
    time.sleep(1)

    # no majority - commit index should be the same
    states = get_nodes_state(HOSTS)
    assert all(s['commitIndex'] == new_log['index'] for s in states.values()), states

from .helpers import *


def test_simple_election():
    states = get_nodes_state(HOSTS)
    leader = get_leader(states)

    assert leader
    assert check_one_leader(states, leader), states.values()

    term = states[leader]['term']

    time.sleep(3)

    states = get_nodes_state(HOSTS)
    new_leader = get_leader(states)
    assert leader == new_leader
    assert term == states[new_leader]['term']


def test_reelection():
    states = get_nodes_state(HOSTS)
    leader = get_leader(states)

    # if the leader disconnects, a new one should be elected
    docker_stop(leader)

    states = get_nodes_state(HOSTS - {leader})
    new_leader = get_leader(states)

    assert new_leader
    assert check_one_leader(states, new_leader), states.values()

    # and after it comes back, he became followe
    docker_start(leader)

    states = get_nodes_state(HOSTS)
    assert get_leader(states) == new_leader
    assert check_one_leader(states, new_leader), states.values()

    # when there is no quorum - there is no leader
    docker_stop(leader)
    docker_stop(new_leader)

    states = get_nodes_state(HOSTS - {leader, new_leader})
    last_leader = get_leader(states)
    docker_stop(last_leader)

    time.sleep(1)

    states = get_nodes_state(HOSTS - {leader, new_leader, last_leader})
    assert not get_leader(states)

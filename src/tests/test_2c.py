from .helpers import *


def test_persistency():
    states = get_nodes_state(HOSTS)
    leader = get_leader(states)

    for i in range(10):
        submit_log_entry(leader, str(i))

    time.sleep(2)

    for host in HOSTS:
        docker_stop(host)

    for host in HOSTS:
        docker_start(host)

    time.sleep(10)

    states = get_nodes_state(HOSTS)
    leader = get_leader(states)
    check_one_leader(states, leader)
    assert all(s['commitIndex'] == 10 for s in states.values()), states

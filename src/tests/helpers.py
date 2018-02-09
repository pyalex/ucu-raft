import json
import requests
import pytest
import subprocess
import time


HOSTS = {'goraft_raft{}_1'.format(i) for i in range(1, 6)}


@pytest.fixture(autouse=True, scope='function')
def docker():
    subprocess.call(['rm', '-rf', 'state'])
    subprocess.call(['docker-compose', 'up', '-d'])

    time.sleep(2)
    yield

    subprocess.call('docker-compose logs | sort -t "|" -k +2d', shell=True)
    subprocess.call(['docker-compose', 'down'])


def docker_start(host):
    print(['docker', 'start', host])
    subprocess.call(['docker', 'start', host])
    time.sleep(2)


def docker_stop(host):
    print(['docker', 'stop', host])
    subprocess.call(['docker', 'stop', host])


def docker_restart(host):
    docker_stop(host)
    docker_start(host)


def find_external_api_port(host):
    output = subprocess.check_output(["docker", "inspect", host])
    container = json.loads(output)
    return container[0]['NetworkSettings']['Ports']['8080/tcp'][0]['HostPort']


def get_nodes_state(nodes):
    return {host: requests.get(
        'http://{host}:{port}/state'.format(host='localhost', port=find_external_api_port(host))
    ).json() for host in nodes}


def get_leader(states):
    return next((node for (node, s) in states.items() if s['state'] == 'Leader'), '')


def check_one_leader(states, leader):
    return all(state['state'] != 'Leader' for host, state in states.items() if host != leader)


def submit_log_entry(leader, log):
    port = find_external_api_port(leader)
    r = requests.post(
        'http://{host}:{port}/new/{log}'.format(host='localhost', port=port, log=log)
    )
    assert r.status_code == 200
    return requests.get('http://{host}:{port}/logs'.format(host='localhost', port=port)).json()['logs'][-1]


def check_log_submitted(nodes, index, command, term):
    logs = [(host, l) for host in nodes for l in get_logs(host)]
    print(logs)
    return [(h, l) for h, l in logs if command in l['command'] and l['index'] == index and l['term'] == term]


def get_common_log_ids(nodes):
    nodes = list(nodes)
    common_ids = set(l['index'] for l in get_logs(nodes[0]))
    for node in nodes[1:]:
        common_ids.intersection_update(set(l['index'] for l in get_logs(node)))

    return common_ids


def check_logs_equal(nodes):
    nodes = list(nodes)
    ids = set(l['index'] for l in get_logs(nodes[0]))
    for node in nodes[1:]:
        node_ids = set(l['index'] for l in get_logs(node))
        if node_ids - ids:
            return False

    return True


def partition(host, targets):
    for target in targets:
        subprocess.call(["docker", "exec", "--", host, "route", "add", "-host", target, "reject"])


def heal_partition(host, targets):
    for target in targets:
        subprocess.call(["docker", "exec", "--", host, "route", "delete", "-host", target, "reject"])


def get_logs(host):
    return requests.get('http://{host}:{port}/logs'.format(
            host='localhost', port=find_external_api_port(host))).json()['logs']

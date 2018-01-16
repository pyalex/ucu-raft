import json
import requests
import pytest
import subprocess
import time


HOSTS = {'goraft_raft{}_1'.format(i) for i in range(1, 6)}


@pytest.fixture(autouse=True, scope='function')
def docker():
    subprocess.call(['docker-compose', 'up', '-d'])

    time.sleep(2)
    yield
    subprocess.call(['docker-compose', 'down'])


def docker_start(host):
    subprocess.call(['docker', 'start', host])
    time.sleep(2)


def docker_stop(host):
    subprocess.call(['docker', 'stop', host])


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
    return all(state['state'] == 'Follower' for host, state in states.items() if host != leader)


def submit_log_entry(leader, log):
    port = find_external_api_port(leader)
    r = requests.post(
        'http://{host}:{port}/new/{log}'.format(host='localhost', port=port, log=log)
    )
    assert r.status_code == 200
    return requests.get('http://{host}:{port}/logs'.format(host='localhost', port=port)).json()['logs'][-1]


def check_log_submitted(nodes, index, command, term):
    logs = [
        (host, l) for host in nodes
        for l in requests.get('http://{host}:{port}/logs'.format(
            host='localhost', port=find_external_api_port(host))).json()['logs']]
    print(logs)
    return [(h, l) for h, l in logs if command in l['command'] and l['index'] == index and l['term'] == term]

from os import listdir
from os.path import isfile, join, exists
import sys
import json
import matplotlib.pyplot as plt
from matplotlib.ticker import PercentFormatter
import numpy as np
import math
from matplotlib.pyplot import figure

# Ensure this matches the variable PerformanceDataVersion at the
# top of coordinator/testruns/testruns.go
version = 4

one_sec = 10**9

def plot_system_mem_available(cmdID, perfdata):
    samples = perfdata['samples']
    fig, (ax) = plt.subplots(nrows=1)
    zero_time = samples[0]['start']
    x = []
    y = {'avail':[],'free':[],'used':[],'shared':[],'buffcache':[]}

    for sample in samples:
        x.append((sample['start']-zero_time) / one_sec)
        y['avail'].append(sample['system_memory']['available'] / 1024)
        y['free'].append(sample['system_memory']['free'] / 1024)
        y['used'].append(sample['system_memory']['used'] / 1024)
        y['shared'].append(sample['system_memory']['shared'] / 1024)
        y['buffcache'].append(sample['system_memory']['buffcache'] / 1024)

    ax.plot(x, y['avail'], label='Available')
    ax.plot(x, y['free'], label='Free')
    ax.plot(x, y['used'], label='Used')
    ax.plot(x, y['shared'], label='Shared')
    ax.plot(x, y['buffcache'], label='Buffer/cache')

    ax.set_ylabel('Memory (MB)')
    ax.set_xlabel('Time since start of command (sec)')
    ax.set_title('Memory stats over time')
    ax.legend()

    plt.tight_layout(rect=[0, 0, 1, 0.95])
    plt.savefig('plots/perf_system_memory_{}_{}.svg'.format(version, cmdID))
    plt.close('all')

def plot_num_threads(cmdID, perfdata):
    samples = perfdata['samples']
    fig, (ax) = plt.subplots(nrows=1)
    zero_time = samples[0]['start']
    x = []
    y = []

    num_samples = 0
    for sample in samples:
        num_samples = num_samples + 1
        x.append((sample['start']-zero_time) / one_sec)
        if 'child_stat' in sample:
            y.append(sample['child_stat']['num_threads'])
        else:
            y.append(0)

    ax.plot(x, y)

    ax.set_ylabel('Threads')
    ax.set_xlabel('Time since start of command (sec)')
    ax.set_title('Number of threads in monitored process over time')

    plt.tight_layout(rect=[0, 0, 1, 0.95])
    plt.savefig('plots/perf_num_threads_{}_{}.svg'.format(version, cmdID))
    plt.close('all')

def plot_process_cpu_usage(cmdID, perfdata):
    samples = perfdata['samples']
    fig, (ax) = plt.subplots(nrows=1)
    zero_time = samples[0]['start']
    x = []
    y = []
    prev_y = -1
    prev_time = -1
    for sample in samples:
        x.append((sample['start']-zero_time) / one_sec)

        clk_tck_diff = (sample['start'] - prev_time) / one_sec * perfdata['clk_tck']
        if 'child_stat' in sample:
            if prev_y > -1:
                y.append((sample['child_stat']['utime'] + sample['child_stat']['stime'] - prev_y) / clk_tck_diff * 100)
            prev_y = sample['child_stat']['utime'] + sample['child_stat']['stime']
            prev_time = sample['start']
        else:
            y.append(0)

    x = x[1:]

    fig, ax = plt.subplots(nrows=1)
    fig.set_size_inches(5,5)
    ax.plot(x,y)
    ax.set_ylabel('CPU Usage (%)')
    ax.set_xlabel('Time since start of command (sec)')
    ax.set_title('Monitored process CPU usage')

    plt.tight_layout(rect=[0, 0, 1, 0.95])
    plt.savefig('plots/perf_process_cpu_usage_{}_{}.svg'.format(version, cmdID), dpi=200)


    plt.close('all')
    
def plot_process_disk_usage(cmdID, perfdata):
    samples = perfdata['samples']
    fig, (ax) = plt.subplots(nrows=1)
    zero_time = samples[0]['start']
    x = []
    y = []

    for sample in samples:
        x.append((sample['start']-zero_time) / one_sec)
        y.append(sample['disk_process']['free'] / 1024)

    print(y)

    ax.plot(x, y)

    ax.set_ylabel('Available Disk Space (MB)')
    ax.set_xlabel('Time since start of command (sec)')
    ax.set_title('Disk space available to process')
    ax.ticklabel_format(useOffset=False)
    
    plt.tight_layout(rect=[0, 0, 1, 0.95])
    plt.savefig('plots/perf_process_disk_usage_{}_{}.svg'.format(version, cmdID))
    plt.close('all')

def plot_cpu_usage(cmdID, perfdata):
    samples = perfdata['samples']
    fig, (ax) = plt.subplots(nrows=1)
    zero_time = samples[0]['start']
    x = []
    prev_y = {'user':-1, 'nice': -1, 'system': -1}
    y = {}

    for sample in samples:
        x.append((sample['start']-zero_time) / one_sec)
        for cpu in sample['cpu_stat']['per_cpu']:
            key = 'cpu{}'.format(cpu['cpu'])
            if key not in y:
                y[key] = {'cpu' : cpu['cpu'], 'prev_time':-1, 'prev_user':-1, 'prev_nice':-1, 'prev_system':-1, 'user':[], 'nice':[], 'system':[]}

            if y[key]['prev_user'] > -1:
                clk_tck_diff = (sample['start'] - y[key]['prev_time']) / one_sec * perfdata['clk_tck']
                y[key]['user'].append((cpu['v'][0] - y[key]['prev_user']) / clk_tck_diff * 100)
                y[key]['nice'].append((cpu['v'][1] - y[key]['prev_nice']) / clk_tck_diff * 100)
                y[key]['system'].append((cpu['v'][2] - y[key]['prev_system']) / clk_tck_diff * 100)

            y[key]['prev_user'] = cpu['v'][0]
            y[key]['prev_nice'] = cpu['v'][1]
            y[key]['prev_system'] = cpu['v'][2]
            y[key]['prev_time'] = sample['start']

    x = x[1:]

    cols = math.ceil(math.sqrt(len(y.keys())))
    rows = math.ceil(len(y.keys()) / cols)

    fig, axs = plt.subplots(nrows=rows, ncols=cols, squeeze=False)
    fig.set_size_inches(5 * cols, 5 * rows)
    i = 0
    for cpu in y.keys():
        col = i % cols
        row = math.floor((i - (i % cols)) / cols)
        axs[row,col].plot(x, y[cpu]['user'], label="User processes")
        axs[row,col].plot(x, y[cpu]['nice'], label="Niced processes")
        axs[row,col].plot(x, y[cpu]['system'], label="System processes")
        axs[row,col].set_ylabel('CPU Usage (%)')
        axs[row,col].set_xlabel('Time since start of command (sec)')
        axs[row,col].set_title('Processor usage (CPU {})'.format(y[cpu]['cpu']))
        axs[row,col].legend()
        i = i + 1

    for j in range(i,cols*rows):
        col = j % cols
        row = math.floor((j - (j % cols)) / cols)
        axs[row,col].axis('off')

    plt.tight_layout(rect=[0, 0, 1, 0.95])
    plt.savefig('plots/perf_cpu_usage_{}_{}.svg'.format(version, cmdID), dpi=200)

    plt.close('all')

def plot_disk_usage(cmdID, perfdata):
    samples = perfdata['samples']
    fig, (ax) = plt.subplots(nrows=1)
    zero_time = samples[0]['start']
    x = []
    y = {}

    for sample in samples:
        x.append((sample['start']-zero_time) / one_sec)
        for disk in sample['disk_all']:
            key = 'disk{}'.format(disk['mountpoint'])
            if key not in y:
                y[key] = {'mountpoint' : disk['mountpoint'], 'total':[], 'used':[], 'free':[]}

            y[key]['total'].append(disk['total'] / 1024)
            y[key]['used'].append(disk['used'] / 1024)
            y[key]['free'].append(disk['free'] / 1024)

    cols = math.ceil(math.sqrt(len(y.keys())))
    rows = math.ceil(len(y.keys()) / cols)

    fig, axs = plt.subplots(nrows=rows, ncols=cols, squeeze=False)
    fig.set_size_inches(5 * cols, 5 * rows)
    i = 0
    for mountpoint in y.keys():
        col = i % cols
        row = math.floor((i - (i % cols)) / cols)
        axs[row,col].plot(x, y[mountpoint]['free'])
        axs[row,col].set_ylabel('Available disk space (MB)')
        axs[row,col].set_xlabel('Time since start of command (sec)')
        axs[row,col].set_title('Available disk space (Mountpoint {})'.format(y[mountpoint]['mountpoint']))
        axs[row,col].ticklabel_format(useOffset=False)

        i = i + 1

    for j in range(i,cols*rows):
        col = j % cols
        row = math.floor((j - (j % cols)) / cols)
        axs[row,col].axis('off')

    plt.tight_layout(rect=[0, 0, 1, 0.95])
    plt.savefig('plots/perf_disk_usage_{}_{}.svg'.format(version, cmdID), dpi=200)

    plt.close('all')

def plot_network_buffers(cmdID, perfdata):
    samples = perfdata['samples']
    zero_time = samples[0]['start']
    x = []
    y = {}

    for sample in samples:
        x.append((sample['start']-zero_time) / one_sec)
        samplerow = []
        for buf in sample['network_buffers']:
            key = None
            if 'remote' in buf and 'localport' in buf:
                key = '{}_{}'.format(buf['remote'], buf['localport'])
            elif 'remote_ip' in buf and 'localport' in buf:
                key = '{}_{}'.format(buf['remote_ip'], buf['localport'])

            if key is not None:
                if key not in samplerow:
                    if key not in y:
                        y[key] = {'s':np.zeros(len(x)-1),'r':np.zeros(len(x)-1)}
                    y[key]['s'] = np.append(y[key]['s'], buf['sendqueue'])
                    y[key]['r'] = np.append(y[key]['r'], buf['recvqueue'])
                    samplerow.append(key)

    for remote in y.keys():
        y[remote]['s'] = np.append(y[remote]['s'], np.zeros(len(x)-len(y[remote]['s'])))
        y[remote]['r'] = np.append(y[remote]['r'], np.zeros(len(x)-len(y[remote]['r'])))

    cols = math.ceil(math.sqrt(len(y.keys())))
    rows = math.ceil(len(y.keys()) / cols)

    fig, axs = plt.subplots(nrows=rows, ncols=cols, squeeze=False)
    fig.set_size_inches(5 * cols, 5 * rows)
    i = 0
    for remote in y.keys():
        col = i % cols
        row = math.floor((i - (i % cols)) / cols)
        axs[row,col].plot(x, y[remote]['s'], label="Send queue")
        axs[row,col].plot(x, y[remote]['r'], label="Recv queue")
        axs[row,col].set_ylabel('Queue size (packets)')
        axs[row,col].set_xlabel('Time since start of command (sec)')
        axs[row,col].set_title('Network queue size over time ({})'.format(remote))
        axs[row,col].legend()
        i = i + 1

    for j in range(i,cols*rows):
        col = j % cols
        row = math.floor((j - (j % cols)) / cols)
        axs[row,col].axis('off')

    plt.tight_layout(rect=[0, 0, 1, 0.95])
    plt.savefig('plots/perf_network_buffers_{}_{}.svg'.format(version, cmdID), dpi=200)

    plt.close('all')



def trim_and_split(line):
    line = line.strip().replace('\t',' ')
    while '  ' in line:
        line = line.replace('  ',' ')

    return line.split(' ')

def process_performance_data(file, config, plot_type):
    cmdID = file[5:].replace('.txt','')
    
    perfdata_file = join('performanceprofiles', 'perf{}_{}.json'.format(version,cmdID))
    
    perfdata = {}
    if not exists(perfdata_file):
        results = {}
    
        perfdata['clk_tck'] = 0.00
        perfdata['page_size'] = 0.00
        perfdata['samples'] = []
        cur_sample = {}
        cur_mode = 0
    
        with open(join('performanceprofiles', file)) as f:
            for line in f:
                if '%CLK_TCK' in line:
                    start_val = line.find('%CLK_TCK') + 9
                    end_val = min(line.find(i, start_val) for i in ['%','\n'] if i in line)
                    if end_val <= 0:
                        end_val = len(line)
                    perfdata['clk_tck'] = float(line[start_val:end_val])
    
                if '%PAGESIZE' in line:
                    start_val = line.find('%PAGESIZE') + 10
                    end_val = min(line.find(i, start_val) for i in ['%','\n'] if i in line)
                    if end_val <= 0:
                        end_val = len(line)
                    perfdata['page_size'] = float(line[start_val:end_val])
    
                if '%SAMPLE_END' in line:
                    start_timestamp = line.index('%SAMPLE_END') + 12
                    cur_sample['end'] = int(line[start_timestamp:start_timestamp+19])
                    perfdata['samples'].append(cur_sample)
                    cur_sample = {}
                    cur_mode = 0
                if '%SAMPLE_START' in line:
                    start_timestamp = line.index('%SAMPLE_START') + 14
                    cur_sample['start'] = int(line[start_timestamp:start_timestamp+19])
                    cur_mode = 0
    
                if line[0:5] == '%FREE':
                    cur_mode = 1
                elif line[0:13] == '%NETBUF-CHILD':
                    cur_sample['network_buffers'] = []
                    cur_mode = 2
                elif line[0:12] == '%STATM-CHILD':
                    cur_mode = 3
                elif line[0:12] == '%STATM-AGENT':
                    cur_mode = 4
                elif line[0:11] == '%STAT-CHILD':
                    cur_mode = 5
                elif line[0:11] == '%STAT-AGENT':
                    cur_mode = 6
                elif line[0:5] == '%STAT':
                    cur_sample['cpu_stat'] = {}
                    cur_sample['cpu_stat']['per_cpu'] = []
                    cur_mode = 7
                elif line[0:7] == '%UPTIME':
                    cur_mode = 8
                elif line[0:8] == '%DISKENV':
                    cur_mode = 9
                elif line[0:8] == '%DISKALL':
                    cur_sample['disk_all'] = []
                    cur_mode = 10
                elif line[0:5] == '%END-':
                    cur_mode = 0
    
                if (cur_mode == 9 or cur_mode == 10) and (not line[0] == '%') and len(line) > 1 and not (line[0:10] == 'Filesystem'):
                    parts = trim_and_split(line)
                    disk_sample = {}
                    disk_sample['mountpoint'] = parts[len(parts)-1]
                    disk_sample['total'] = int(parts[len(parts)-5])
                    disk_sample['used'] = int(parts[len(parts)-4])
                    disk_sample['free'] = int(parts[len(parts)-3])
                    
                    if cur_mode == 9:
                        print(disk_sample)
                        cur_sample['disk_process'] = disk_sample
                    if cur_mode == 10:
                        cur_sample['disk_all'].append(disk_sample)
    
                if line[0:4] == 'Mem:' and cur_mode == 1:
                    parts = trim_and_split(line)
                    cur_sample['system_memory'] = {}
                    cur_sample['system_memory']['total'] = int(parts[1])
                    cur_sample['system_memory']['used'] = int(parts[2])
                    cur_sample['system_memory']['free'] = int(parts[3])
                    cur_sample['system_memory']['shared'] = int(parts[4])
                    cur_sample['system_memory']['buffcache'] = int(parts[5])
                    cur_sample['system_memory']['available'] = int(parts[6])
    
                if cur_mode == 2 and (not line[0] == '%') and len(line) > 1:
                    parts = trim_and_split(line)
    
                    netbuf = {}
                    netbuf['recvqueue'] = int(parts[1])
                    netbuf['sendqueue'] = int(parts[2])
                    netbuf['localport'] = int(parts[3][parts[3].index(':')+1:])
                    remote_ip = parts[4][:parts[4].index(':')]
                    netbuf['remote_ip'] = remote_ip
                    for key in config:
                        if config[key] == remote_ip:
                            netbuf['remote'] = key[:len(key)-3]
    
                    cur_sample['network_buffers'].append(netbuf)
    
                if (cur_mode == 5 or cur_mode == 6) and (not line[0] == '%') and not(line.strip() == ''):
                    parts = trim_and_split(line)
                    cpu_stat = {}
                    cpu_stat['state'] = parts[2]
                    cpu_stat['utime'] = int(parts[13])
                    cpu_stat['stime'] = int(parts[14])
                    cpu_stat['cutime'] = int(parts[15])
                    cpu_stat['cstime'] = int(parts[16])
                    cpu_stat['num_threads'] = int(parts[19])
                    cpu_stat['vsize'] = int(parts[22])
                    cpu_stat['rss'] = int(parts[23])
                    cpu_stat['start'] = int(parts[21])
    
                    if cur_mode == 5:
                        cur_sample['child_stat'] = cpu_stat
                    if cur_mode == 6:
                        cur_sample['agent_stat'] = cpu_stat
    
                if cur_mode == 7:
                    parts = trim_and_split(line)
                    if parts[0] == 'cpu':
                        cur_sample['cpu_stat']['total_cpu'] = [int(n) for n in parts[1:]]
                    elif parts[0][0:3] == 'cpu':
                        per_cpu = {}
                        per_cpu['cpu'] = int(parts[0][3:])
                        per_cpu['v'] = [int(n) for n in parts[1:]]
                        cur_sample['cpu_stat']['per_cpu'].append(per_cpu)
                    else:
                        cur_sample['cpu_stat'][parts[0]] = [int(n) for n in parts[2:]]
    
        with open(perfdata_file, 'w') as outfile:
            json.dump(perfdata, outfile)
    else:
        with open(perfdata_file, 'r') as infile:
            perfdata = json.load(infile)

    if plot_type == "system_memory":
        plot_system_mem_available(cmdID, perfdata)
    elif plot_type == "network_buffers":
        plot_network_buffers(cmdID, perfdata)
    elif plot_type == "cpu_usage":
        plot_cpu_usage(cmdID, perfdata)
    elif plot_type == "num_threads":
        plot_num_threads(cmdID, perfdata)
    elif plot_type == "process_cpu_usage":
        plot_process_cpu_usage(cmdID, perfdata)
    elif plot_type == "process_disk_usage":
        plot_process_disk_usage(cmdID, perfdata)
    elif plot_type == "disk_usage":
        plot_disk_usage(cmdID, perfdata)


config_files = []
if exists('outputs'):
    config_files = [f for f in listdir('outputs') if isfile(join('outputs', f)) and 'config.cfg' in f]

config = {}
if len(config_files) > 0:
    with open(join('outputs', config_files[0])) as f:
        for line in f:
            line = line.strip()
            if '=' in line:
                split = line.index('=')
                key = line[0:split]
                value = line[split+1:]
                if value[0] == '"':
                    value = value[1:len(value)-1]

                config[key] = value

append = {}

for key in config:
    if '_endpoint' in key:
        append[key[0:len(key)-9] + '_ip'] = config[key][:config[key].index(':')]

for key in append:
    config[key] = append[key]

output_files = [f for f in listdir('performanceprofiles') if isfile(join('performanceprofiles', f)) and '.txt' in f and sys.argv[1] in f]
process_performance_data(output_files[0], config, sys.argv[2])
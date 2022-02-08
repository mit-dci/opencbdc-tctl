from os import listdir, environ, remove
from os.path import isfile, join
import random
import sys
import json
import matplotlib.pyplot as plt
from matplotlib.ticker import PercentFormatter
import numpy as np
import vaex
from vaex import BinnerTime
import pandas
import datetime
import matplotlib.dates as mdates

# Ensure this matches the variable TestResultVersion at the
# top of coordinator/testruns/testruns.go
version = 2

class MyBinnerTime(BinnerTime):
    def __init__(self, expression, resolution='W', df=None, every=1):
        self._promise = vaex.promise.Promise.fulfilled(None)
        self.every = every
        self.resolution = resolution
        self.expression = expression
        self.df = df or expression.ds
        self.sort_indices = None
        # make sure it's an expression
        self.expression = self.df[str(self.expression)]
        self.tmin, self.tmax = self.df[str(self.expression)].min(), self.df[str(self.expression)].max()

        self.label = ''

        self.resolution_type = 'M8[%s]' % self.resolution
        dt = (self.tmax.astype(self.resolution_type) - self.tmin.astype(self.resolution_type))
        self.N = (dt.astype(int).item() + 1)
        # divide by every, and round up
        self.N = (self.N + every - 1) // every
        self.bin_values = np.arange(self.tmin.astype(self.resolution_type), self.tmax.astype(self.resolution_type)+1, every)
        # TODO: we modify the dataframe in place, this is not nice
        self.begin_name = self.df.add_variable('t_begin', self.tmin.astype(self.resolution_type), unique=True)
        # TODO: import integer from future?
        self.binby_expression = str(self.df['%s - %s' % (self.expression.astype(self.resolution_type), self.begin_name)].astype('int') // every)
        self.binner = self.df._binner_ordinal(self.binby_expression, self.N)

one_sec = 10**9

def read_block_log(file, global_min_time):
    p = pandas.read_csv(join("outputs", file), sep=' ', error_bad_lines=False, warn_bad_lines=True, names=['time', 'latency', 'height'], encoding="ISO-8859-1")
    if p.size > 0:
        min_time = np.min(p.time)
        if(min_time > global_min_time and global_min_time > 0):
            min_time = global_min_time
        p['lats'] = p.latency // 10**6
        p['latsS'] = p.lats / 10**3
        return p, min_time
    return None, 0

def read_throughput_sample_file(file):
    samples = []
    failed_values = 0
    with open(join("outputs", file), encoding="ISO-8859-1") as f:
        for line in f:
            try:
                val = float(line[:-1])
                samples.append(val)
            except:
                failed_values = failed_values + 1
    if failed_values > 10:
        raise "Too many failed values in throughput file {}".format(file)
    return samples

def read_latency_sample_file(file):
    samples = []
    failed_values = 0
    with open(join("outputs", file), encoding="ISO-8859-1") as f:
        for line in f:
            try:
                val = float(line[:-1])
                samples.append(val/one_sec)
            except:
                failed_values = failed_values + 1
    if failed_values > 10:
        raise "Too many failed values in throughput file {}".format(file)
    return samples

lats = []

output_files = [f for f in listdir('outputs') if isfile(join("outputs", f))]

archiver_based = False
two_phase = False
for output_file in output_files:
    if 'tx_samples' in output_file:
        two_phase = True
    if 'tp_samples' in output_file:
        archiver_based = True

block_time_ms = 1000

if archiver_based:
    if 'BLOCK_TIME' in environ:
        block_time_ms=int(environ['BLOCK_TIME'])
    else:
        config_files = ['outputs/' + x for x in listdir('outputs') if '.cfg' in x]
        with open(config_files[0]) as f:
            for line in f:
                if '=' in line:
                    split = line.index('=')
                    key = line[0:split]
                    value = line[split+1:]
                    if value[0] == '"':
                        value = value[1:len(value)-1]
                    if key == 'target_block_interval':
                        block_time_ms=int(value)

tps_lines = []

if two_phase:
    hdf5_files = ['outputs/' + x for x in listdir('outputs') \
             if 'tx_samples' in x and 'hdf5' in x]

    for hdf in hdf5_files:
        remove(hdf)

    files = ['outputs/' + x for x in listdir('outputs') \
             if 'tx_samples' in x and 'hdf5' not in x]

    for f in files:
        p = pandas.read_csv(f, sep=' ', error_bad_lines=True, warn_bad_lines=True, names=['time', 'latency'], encoding="ISO-8859-1")
        if p.dtypes['time'] != np.int64:
            p.time = pandas.to_numeric(p.time, errors='coerce', downcast='integer')
            p = p[pandas.notnull(p.time)]

        if p.size > 0:
            v = vaex.from_pandas(p, copy_index=False)
            v.export_hdf5(f + '.hdf5')
        else:
            print('{} has no rows', f)

    df = vaex.open('outputs/*.txt.hdf5')
    df['lats'] = df.latency // 10**6
    df['latsS'] = df.lats / 10**3
    df['pDate'] = df.time.values.astype('datetime64[ns]')
    df = df[df.time > 1609459200000] # Filter out (corrupt) times before 2021
    dat = df.groupby(by=MyBinnerTime(expression=df.pDate, resolution='s', df=df), agg={'count': 'count'})
    tps_its = dat.to_items()
    current = tps_its[0][1][0].astype(datetime.datetime)
    end = tps_its[0][1][-1].astype(datetime.datetime)
    tps = []
    idx = 0
    while current < end:
        dt = tps_its[0][1][idx].astype(datetime.datetime)
        if dt != current:
            tps.append(0)
        else:
            tps.append(tps_its[1][1][idx])
            idx += 1
        current += datetime.timedelta(seconds=1)

    dat = df.groupby(df.latsS, agg='count')
    lats = dat.values
    lat_max = df.max(df.latsS)
    lat_min = df.min(df.latsS)
    n_bins = 15
    lats_binsize = (lat_max - lat_min) / n_bins
    bin_edges = []
    bin_depths = []
    for i in range(n_bins):
        bin_edges.append(lat_min + i * lats_binsize)
        bin_depths.append(0)
    bin_edges = np.array(bin_edges)
    current_bin = 0
    sorted_lats = lats[lats[:,0].argsort()]
    for val in sorted_lats:
        if current_bin + 1 < n_bins and val[0] >= bin_edges[current_bin+1]:
            current_bin += 1
        bin_depths[current_bin] += val[1]
    bin_depths /= np.sum(bin_depths)
    tps_lines.append({"tps":tps, "title":"Loadgens", "freq": 1})

if archiver_based:
    for output_file in output_files:
        if output_file.find('tp_samples') > -1:
            filetps = read_throughput_sample_file(output_file)
            tps_lines.append({"tps":filetps, "freq": (block_time_ms/1000), "title": output_file.replace("-tp_samples.txt","")})
        elif not two_phase and output_file.find('latency_samples_') > -1:
            filelats = read_latency_sample_file(output_file)
            lats.extend(filelats)


atomizer_discard_log = ['outputs/' + x for x in listdir('outputs') if 'raft-atomizer' in x and 'discarded_log.txt' in x]
if len(atomizer_discard_log) > 0:
    for f in atomizer_discard_log:
        p = pandas.read_csv(f, sep=' ', error_bad_lines=True, warn_bad_lines=True, names=['reason', 'time'], encoding="ISO-8859-1")
        if p.dtypes['time'] != np.int64:
            p.time = pandas.to_numeric(p.time, errors='coerce', downcast='integer')
            p = p[pandas.notnull(p.time)]

        v = vaex.from_pandas(p, copy_index=False)
        v['pDate'] = v.time.values.astype('datetime64[ns]')
        exp = v[v.reason == 'exp']
        print(exp)
        spent = v[v.reason == 'spe']
        print(spent)
        if len(exp) > 0:
            exp_dat = exp.groupby(by=MyBinnerTime(expression=exp.pDate, resolution='s', df=exp), agg={'count': 'count'})
            exp_tps_its = exp_dat.to_items()
            current = exp_tps_its[0][1][0].astype(datetime.datetime)
            end = exp_tps_its[0][1][-1].astype(datetime.datetime)
            exptps = []
            idx = 0
            while current < end:
                dt = exp_tps_its[0][1][idx].astype(datetime.datetime)
                if dt != current:
                    exptps.append(0)
                else:
                    exptps.append(exp_tps_its[1][1][idx])
                    idx += 1
                current += datetime.timedelta(seconds=1)
            tps_lines.append({"tps":exptps, "title":"Discarded by Atomizer (expired)", "freq": 1})
        if len(spent) > 0:
            spent_dat = spent.groupby(by=MyBinnerTime(expression=spent.pDate, resolution='s', df=spent), agg={'count': 'count'})
            spent_tps_its = spent_dat.to_items()
            current = spent_tps_its[0][1][0].astype(datetime.datetime)
            end = spent_tps_its[0][1][-1].astype(datetime.datetime)
            spenttps = []
            idx = 0
            while current < end:
                dt = spent_tps_its[0][1][idx].astype(datetime.datetime)
                if dt != current:
                    spenttps.append(0)
                else:
                    spenttps.append(spent_tps_its[1][1][idx])
                    idx += 1
                current += datetime.timedelta(seconds=1)
            tps_lines.append({"tps":spenttps, "title":"Discarded by Atomizer (spent)", "freq": 1})


hdf5_files = ['outputs/' + x for x in listdir('outputs') \
             if 'block_log.txt' in x and 'hdf5' in x]

for hdf in hdf5_files:
    remove(hdf)

block_lines = []

global_min_time = 0
for output_file in output_files:
    if 'block_log.txt' in output_file:
        block_data, min_time = read_block_log(output_file, global_min_time)
        if block_data is not None:
            if global_min_time == 0 or global_min_time > min_time:
                global_min_time = min_time
            block_lines.append({"data":block_data, "title":output_file.replace("-block_log.txt","")})

for block_line in block_lines:
    block_line['data']['timeOffset'] = block_line['data']['time'] - global_min_time
    block_line['data']['pDate'] = block_line['data']['timeOffset'].values.astype('datetime64[ns]')

## Lob off zero samples at the start (while the system is started but no
## load is generated yet)

if 'TRIM_ZEROES_START' in environ and environ['TRIM_ZEROES_START'] == "1":
    for i in range(len(tps_lines)):
        while len(tps_lines[i]["tps"]) > 0 and int(tps_lines[i]["tps"][0]) == 0:
            tps_lines[i]["tps"] = tps_lines[i]["tps"][1:]

if 'TRIM_ZEROES_END' in environ and environ['TRIM_ZEROES_END'] == "1":
    for i in range(len(tps_lines)):
       while len(tps_lines[i]["tps"]) > 0 and int(tps_lines[i]["tps"][-1]) == 0:
            tps_lines[i]["tps"].pop()

## Lob off (configurable) more "warm up" samples
if 'TRIM_SAMPLES' in environ:
    trim_samples = int(environ['TRIM_SAMPLES'])
    for i in range(len(tps_lines)):
        tps_lines[i]["tps"] = tps_lines[i]["tps"][trim_samples:]


## Create throughput histogram
fig, (ax) = plt.subplots(nrows=1)

colors = ['blue','red','green','orange','black','purple','cyan']

print(tps_lines)

avg_tp = np.mean(tps_lines[0]["tps"])
sigma_tp = np.std(tps_lines[0]["tps"])

for i, tps_line in enumerate(tps_lines):
    weights = np.ones_like(tps_line["tps"]) / len(tps_line["tps"])
    color = (random.random(), random.random(), random.random())
    if len(colors) > i:
        color = colors[i]

    ax.hist(tps_line["tps"], weights=weights, label=tps_line["title"], bins=15, edgecolor='black', color=color)

ax.yaxis.set_major_formatter(PercentFormatter(xmax=1))

def val_to_dev(val):
    return (val - avg_tp) / sigma_tp

def dev_to_val(dev):
    return (dev * sigma_tp) + avg_tp

std_axis = ax.secondary_xaxis('top', functions=(val_to_dev, dev_to_val))
std_axis.set_xlabel('+/- σ')

ax.set_ylabel('Percentage')
ax.set_xlabel('Throughput (TX/s)')
ax.set_title('Distribution')

if len(tps_lines) > 1:
    ax.legend()

plt.tight_layout(rect=[0, 0, 1, 0.95])
plt.savefig('plots/system_throughput_hist.png')
plt.close('all')
## Create latency histogram

fig, (ax) = plt.subplots(nrows=1)

if two_phase:
    avg_lat = df.mean(df.latsS)
    sigma_lat = df.std(df.latsS)
    ax.bar(bin_edges, bin_depths, width=lats_binsize, edgecolor='black')

if not two_phase:
    weights = np.ones_like(lats) / len(lats)
    avg_lat = np.mean(lats)
    sigma_lat = np.std(lats)
    ax.hist(lats, weights=weights, bins=15, edgecolor='black')

ax.yaxis.set_major_formatter(PercentFormatter(xmax=1))

def val_to_dev(val):
    return (val - avg_lat) / sigma_lat

def dev_to_val(dev):
    return (dev * sigma_lat) + avg_lat

std_axis = ax.secondary_xaxis('top', functions=(val_to_dev, dev_to_val))
std_axis.set_xlabel('+/- σ')

ax.set_ylabel('Percentage')
ax.set_xlabel('Latency (sec)')
ax.set_title('Distribution')

plt.tight_layout(rect=[0, 0, 1, 0.95])
plt.savefig('plots/system_latency_hist.png')
plt.close('all')



## Create throughput line

fig, (ax) = plt.subplots(nrows=1)

max = 0
for i, tps_line in enumerate(tps_lines):
    if len(tps_line["tps"]) == 0: continue
    tps_time = []
    tps_ma = []
    time = 0
    tps_ma_tmp = []
    tps_ma_ms = 5000
    if archiver_based:
        tps_ma_ms = block_time_ms * 5

    for t in tps_lines[i]["tps"]:
        tps_ma_tmp.append(t)
        while len(tps_ma_tmp) > 5:
            tps_ma_tmp = tps_ma_tmp[1:]

        val = np.mean(tps_ma_tmp)
        if max < val:
            max = val
        tps_ma.append(np.mean(tps_ma_tmp))


        time = time + tps_lines[i]["freq"]
        tps_time.append(datetime.datetime.fromtimestamp(time))

    color = (random.random(), random.random(), random.random())
    if len(colors) > i:
        color = colors[i]
    ax.plot(tps_time, tps_ma, label=tps_line["title"], color=color)




ax.set_ylabel('Throughput (TX/s, {}ms moving average)'.format(tps_ma_ms))
ax.set_xlabel('Time (mm:ss)')
ax.set_title('Time series')
ax.set_ylim(ymin=0, ymax=max)
ax.set_xlim(xmin=datetime.datetime.fromtimestamp(0))

timeFmt = mdates.DateFormatter('%M:%S')

ax.xaxis.set_major_formatter(timeFmt)

if len(tps_lines) > 1:
    ax.legend()

plt.tight_layout(rect=[0, 0, 1, 0.95])
plt.grid()
plt.savefig('plots/system_throughput_line.png')
plt.close('all')




## Create blockheight line

fig, (ax) = plt.subplots(nrows=1)
for i, block_line in enumerate(block_lines):
    tps_time = []
    tps_ma = []

    color = (random.random(), random.random(), random.random())
    if len(colors) > i:
        color = colors[i]
    ax.plot(block_line["data"]["pDate"], block_line["data"]["height"], label=block_line["title"], color=color)

ax.set_ylabel('Block height')
ax.set_xlabel('Time (mm:ss)')
ax.set_title('Block height')
ax.set_ylim(ymin=0)
ax.legend()
#ax.set_xlim(xmin=datetime.datetime.fromtimestamp(0))

timeFmt = mdates.DateFormatter('%M:%S')

ax.xaxis.set_major_formatter(timeFmt)

plt.tight_layout(rect=[0, 0, 1, 0.95])
plt.savefig('plots/block_heights_line.png')
plt.close('all')

# Workaround for https://github.com/vaexio/vaex/issues/385
# The first percentile on linux comes out to NaN, so put all the data we want
# after the first index.
buckets = [0.001, 0.001, 0.01, 0.1, 1, 25, 50, 75, 99, 99.9, 99.99, 99.999]
tp_percentiles = np.percentile(tps_lines[0]["tps"], buckets[1:])

if two_phase:
    try:
        # TODO: this shouldn't fail. We should submit a bug report to Vaex
        lat_percentiles = df.percentile_approx(df.latsS, percentage=buckets)[1:]
    except:
        lat_percentiles = np.percentile(df.latsS.tolist(), buckets[1:])

if not two_phase:
    lat_percentiles = np.percentile(lats, buckets[1:])


results = {}
results['blockLatencies'] = {}

for i, block_line in enumerate(block_lines):
    results['blockLatencies'][block_line["title"]] = {}
    results['blockLatencies'][block_line["title"]]['avg'] = np.mean(block_line["data"]["latsS"]).item()
    results['blockLatencies'][block_line["title"]]['min'] = np.min(block_line["data"]["latsS"]).item()
    results['blockLatencies'][block_line["title"]]['max'] = np.max(block_line["data"]["latsS"]).item()
    results['blockLatencies'][block_line["title"]]['std'] = np.std(block_line["data"]["latsS"]).item()


results['throughputAvg'] = np.mean(tps_lines[0]["tps"]).astype(float)
results['throughputMin'] = np.min(tps_lines[0]["tps"]).astype(float)
results['throughputMax'] = np.max(tps_lines[0]["tps"]).astype(float)
results['throughputStd'] = np.std(tps_lines[0]["tps"]).astype(float)
results['throughputPercentiles'] = []

if len(tps_lines) > 1:
    results['throughputAvg2'] = 0
    if len(tps_lines[1]["tps"]) > 0:
        results['throughputAvg2'] = np.mean(tps_lines[1]["tps"]).astype(float)
    results['throughputAvgs'] = {}
    for i, tps_line in enumerate(tps_lines):
        if len(tps_line["tps"]) > 0:
            results['throughputAvgs'][tps_line["title"]] = np.mean(tps_line["tps"]).astype(float)


if two_phase:
    results['latencyAvg'] = df.mean(df.latsS).item()
    results['latencyMin'] = df.min(df.latsS).item()
    results['latencyMax'] = df.max(df.latsS).item()
    results['latencyStd'] = df.std(df.latsS).item()

if not two_phase:
    results['latencyAvg'] = np.mean(lats).astype(float)
    results['latencyMin'] = np.min(lats).astype(float)
    results['latencyMax'] = np.max(lats).astype(float)
    results['latencyStd'] = np.std(lats).astype(float)

results['latencyPercentiles'] = []
for i in range(len(buckets) - 1):
    lat_percentile = {}
    lat_percentile['bucket'] = buckets[1:][i]
    lat_percentile['value'] = lat_percentiles[i].astype(float)
    results['latencyPercentiles'].append(lat_percentile)

    tp_percentile = {}
    tp_percentile['bucket'] = buckets[1:][i]
    tp_percentile['value'] = tp_percentiles[i].astype(float)
    results['throughputPercentiles'].append(tp_percentile)

with open('results{}.json'.format(version), 'w') as outfile:
    json.dump(results, outfile, allow_nan=False)

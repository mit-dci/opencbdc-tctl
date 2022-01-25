from os import listdir, environ
from os.path import isfile, join
from itertools import groupby
from operator import itemgetter
import sys
import json
import math
import matplotlib.pyplot as plt
from matplotlib.ticker import PercentFormatter
from matplotlib import container
import matplotlib.colors as mcolors
import numpy as np
import vaex
import pandas

def set_plot_limits(ax, cfg, idx = ''):
    xmax = None
    ymax = None
    xmin = None
    ymin = None


    if 'x' + idx + 'Max' in cfg['request']:
        xmax = cfg['request']['x' + idx + 'Max']
    if 'y' + idx + 'Max' in cfg['request']:
        ymax = cfg['request']['y' + idx + 'Max']
    if 'x' + idx + 'Min' in cfg['request']:
        xmin = cfg['request']['x' + idx + 'Min']
    if 'y' + idx + 'Min' in cfg['request']:
        ymin = cfg['request']['y' + idx + 'Min']

    if xmax == -1:
        xmax = None
    if ymax == -1:
        ymax = None
    if xmin == -1:
        xmin = None
    if ymin == -1:
        ymin = None

    ax.set_ylim(ymin=ymin, ymax=ymax)
    ax.set_xlim(xmin=xmin, xmax=xmax)

def line_plot(cfg):
    fig, (ax) = plt.subplots(nrows=1)
    fig.set_size_inches(5,5)

    marker = '^'
    if 'pointStyle' in cfg['request']:
        marker = cfg['request']['pointStyle']
    marker2 = 's'
    if 'pointStyle2' in cfg['request']:
        marker2 = cfg['request']['pointStyle2']

    colors = [[mcolors.CSS4_COLORS['blue'], mcolors.CSS4_COLORS['aqua'], mcolors.CSS4_COLORS['springgreen'], mcolors.CSS4_COLORS['forestgreen'], mcolors.CSS4_COLORS['violet'], mcolors.CSS4_COLORS['purple']], [mcolors.CSS4_COLORS['red'], mcolors.CSS4_COLORS['darkred'], mcolors.CSS4_COLORS['peru'], mcolors.CSS4_COLORS['orange'], mcolors.CSS4_COLORS['grey'], mcolors.CSS4_COLORS['gold']]]

    series = {}
    multiX = False
    multiY = False
    for r in cfg['data']:
        serieVal = -1
        if 'seriesFieldEval' in cfg['request']:
            serieVal = eval(cfg['request']['seriesFieldEval'])

        if serieVal not in series:
            series[serieVal] = {'x':[], 'x2': [], 'y':[], 'y2':[]}

        xVal = eval(cfg['request']['xFieldEval'])
        yVal = eval(cfg['request']['yFieldEval'])
        series[serieVal]['x'].append(xVal)
        series[serieVal]['y'].append(yVal)
        if 'y2FieldEval' in cfg['request']:
            multiY = True
            y2Val = eval(cfg['request']['y2FieldEval'])
            series[serieVal]['y2'].append(y2Val)

        if 'x2FieldEval' in cfg['request']:
            multiX = True
            x2Val = eval(cfg['request']['x2FieldEval'])
            series[serieVal]['x2'].append(x2Val)

    xColor = mcolors.BASE_COLORS['k']
    yColor = mcolors.BASE_COLORS['k']
    if multiY:
        yColor = colors[0][0]
        ax2 = ax.twinx()
        ax2.set_ylabel(cfg['request']['y2FieldName'])
        #ax2.tick_params(axis='y')

    if multiX:
        xColor = colors[0][0]
        ax2 = ax.twiny()
        ax2.set_xlabel(cfg['request']['x2FieldName'])
        #ax2.tick_params(axis='x')

    ax.set_xlabel(cfg['request']['xFieldName'])
    ax.set_ylabel(cfg['request']['yFieldName'])
    #ax.tick_params(axis='y', labelcolor=yColor)
    #ax.tick_params(axis='x', labelcolor=xColor)
    ax.set_title(cfg['request']['title'])

    idx = 0
    lns = []
    for serieVal in sorted(series):
        x = series[serieVal]['x']
        x2 = series[serieVal]['x2']
        y = series[serieVal]['y']
        y2 = series[serieVal]['y2']

        if len(y2) > 0:
            y = zip(y, y2)
        if len(x2) > 0:
            x = zip(x, x2)
        x, y = (list(t) for t in zip(*sorted(zip(x, y))))

        if ('groupByX' in cfg['request']) and cfg['request']['groupByX']:
            newX = []
            newY = []
            for groupedX, groupedY in groupby(zip(x,y), itemgetter(0)):
                newX.append(groupedX)
                yVals = list(list(zip(*groupedY))[1])
                newY.append(eval(cfg['request']['groupByXEval']))

            x = newX
            y = newY

        if len(y2) > 0:
            label = '{} ({})'.format(cfg['request']['yFieldNameShorthand'],serieVal)
            if len(series) == 1:
                label = cfg['request']['yFieldNameShorthand']
            ax.plot(x, [val[0] for val in y], color=colors[0][idx], marker=marker, label=label)
        elif len(x2) > 0:
            label = '{} ({})'.format(cfg['request']['yFieldNameShorthand'],serieVal)
            if len(series) == 1:
                label = cfg['request']['yFieldNameShorthand']
            ax.plot([val[0] for val in x], y, color=colors[0][idx], marker=marker, label=label)
        else:
            ax.plot(x, y, color=colors[0][idx], marker=marker, label='{}'.format(serieVal))


        if len(y2) > 0:
            label = '{} ({})'.format(cfg['request']['y2FieldNameShorthand'],serieVal)
            if len(series) == 1:
                label = cfg['request']['y2FieldNameShorthand']
            ax2.plot(x, [val[1] for val in y], color=colors[1][idx], marker=marker2, label=label)
        elif len(x2) > 0:
            label = '{} ({})'.format(cfg['request']['yFieldNameShorthand'],serieVal)
            if len(series) == 1:
                label = cfg['request']['yFieldNameShorthand']
            ax2.plot([val[1] for val in x], y, color=colors[1][idx], marker=marker2, label=label)

        idx = idx + 1

    if multiX or multiY:
        set_plot_limits(ax2, cfg, '2')

    lns, labs = ax.get_legend_handles_labels()
    if multiX or multiY:
        lns2, labs2 = ax2.get_legend_handles_labels()
        lns = lns + lns2
        labs = labs + labs2

    lgd = None
    if (multiX or multiY) or len(series) > 1:
        lns = [h[0] if isinstance(h, container.ErrorbarContainer) else h for h in lns]
        title = None
        if not (multiX or multiY):
            title = cfg['request']['seriesFieldName']
        lgd = ax.legend(lns, labs, title=title, bbox_to_anchor=(0.5, -0.1), loc='upper center')

    set_plot_limits(ax, cfg)
    #plt.tight_layout(rect=[0, 0, 1.0, 0.95])
    if 'xScale' in cfg['request']:
        plt.xscale(cfg['request']['xScale'])
    if 'yScale' in cfg['request']:
        plt.yscale(cfg['request']['yScale'])
    plt.grid()
    ea = None
    if lgd is not None:
        ea = (lgd,)
    plt.savefig(sys.argv[2], bbox_extra_artists=ea, bbox_inches='tight', dpi=200)
    plt.close('all')

def line_plot_err(cfg):
    fig, (ax) = plt.subplots(nrows=1)
    fig.set_size_inches(5,5)

    marker = '^'
    if 'pointStyle' in cfg['request']:
        marker = cfg['request']['pointStyle']

    marker2 = 's'
    if 'pointStyle2' in cfg['request']:
        marker2 = cfg['request']['pointStyle2']

    colors = [[mcolors.CSS4_COLORS['blue'], mcolors.CSS4_COLORS['aqua'], mcolors.CSS4_COLORS['springgreen'], mcolors.CSS4_COLORS['forestgreen'], mcolors.CSS4_COLORS['violet'], mcolors.CSS4_COLORS['purple']], [mcolors.CSS4_COLORS['red'], mcolors.CSS4_COLORS['darkred'], mcolors.CSS4_COLORS['peru'], mcolors.CSS4_COLORS['orange'], mcolors.CSS4_COLORS['grey'], mcolors.CSS4_COLORS['gold']]]
    series = {}
    multiAxis = False

    for rr in cfg['data']:
        r = rr

        serieVal = -1
        if 'seriesFieldEval' in cfg['request']:
            serieVal = eval(cfg['request']['seriesFieldEval'])

        if serieVal not in series:
            series[serieVal] = {'x':[], 'ySets':[], 'y2Sets':[]}

        xVal = eval(cfg['request']['xFieldEval'])
        yVals = []
        for rd in rr['resultDetails']:
            r = {'config':rr['config'], 'result': rd}
            yVals.append(eval(cfg['request']['yFieldEval']))

        series[serieVal]['x'].append(xVal)
        series[serieVal]['ySets'].append(yVals)

        if 'y2FieldEval' in cfg['request']:
            multiAxis = True
            y2Vals = []
            for rd in rr['resultDetails']:
                r = {'config':rr['config'], 'result': rd}
                y2Vals.append(eval(cfg['request']['y2FieldEval']))
            series[serieVal]['y2Sets'].append(y2Vals)


    xColor = mcolors.BASE_COLORS['k']
    yColor = mcolors.BASE_COLORS['k']

    if multiAxis:
        yColor = colors[0][0]

    ax.set_xlabel(cfg['request']['xFieldName'])
    ax.set_ylabel(cfg['request']['yFieldName'])
    ax.set_title(cfg['request']['title'])
    ax.tick_params(axis='y')
    ax.tick_params(axis='x')
    if multiAxis:
        ax2 = ax.twinx()

    idx = 0
    for serieVal in sorted(series):
        maxLenY = 0
        maxLenY2 = 0
        if multiAxis:
            series[serieVal]['ySets'] = zip(series[serieVal]['ySets'], series[serieVal]['y2Sets'])

        x, ySets = (list(t) for t in zip(*sorted(zip(series[serieVal]['x'], series[serieVal]['ySets']))))

        for ySet in ySets:
            if multiAxis:
                if len(ySet[0]) > maxLenY:
                    maxLenY = len(ySet[0])
                if len(ySet[1]) > maxLenY2:
                    maxLenY2 = len(ySet[1])
            else:
                if len(ySet) > maxLenY:
                    maxLenY = len(ySet)

        y = []
        yerr = []
        y2 = []
        y2err = []
        for ySet in ySets:
            setLen = len(ySet)
            if multiAxis:
                setLen = len(ySet[0])

            if setLen / maxLenY * 100 < cfg['request']['successThreshold']:
                y.append(np.nan)
                yerr.append(np.nan)
            elif multiAxis:
                y.append(np.mean(ySet[0]))
                yerr.append(np.std(ySet[0]) / math.sqrt(len(ySet[0])))
            else:
                y.append(np.mean(ySet))
                yerr.append(np.std(ySet) / math.sqrt(len(ySet)))

        if multiAxis:
            for ySet in ySets:
                if len(ySet[1]) / maxLenY2 * 100 < cfg['request']['successThreshold']:
                    y2.append(np.nan)
                    y2err.append(np.nan)
                else:
                    y2.append(np.mean(ySet[1]))
                    y2err.append(np.std(ySet[1]) / math.sqrt(len(ySet[1])))

        if multiAxis:
            label = '{} ({})'.format(cfg['request']['yFieldNameShorthand'],serieVal)
            if len(series) == 1:
                label = cfg['request']['yFieldNameShorthand']
        else:
            label = serieVal

        ax.errorbar(x, y, yerr=yerr, capsize=4.0, color=colors[0][idx], marker=marker, label=label)
        if len(y2) > 0:
            label = '{} ({})'.format(cfg['request']['y2FieldNameShorthand'],serieVal)
            if len(series) == 1:
                label = cfg['request']['y2FieldNameShorthand']
            ax2.errorbar(x, y2, yerr=y2err, capsize=4.0, color=colors[1][idx], marker=marker2, label=label)
            ax2.set_ylabel(cfg['request']['y2FieldName'])
            ax2.tick_params(axis='y')

        idx = idx + 1

    lgd = None
    if len(series) > 0:
        lns, labs = ax.get_legend_handles_labels()
        if len(y2) > 0:
            lns2, labs2 = ax2.get_legend_handles_labels()
            lns = lns + lns2
            labs = labs + labs2

        lns = [h[0] if isinstance(h, container.ErrorbarContainer) else h for h in lns]
        title = None
        if not multiAxis:
            title = cfg['request']['seriesFieldName']
        lgd = ax.legend(lns, labs, title=title, bbox_to_anchor=(0.5, -0.1), loc='upper center')



    set_plot_limits(ax, cfg)
    if multiAxis:
        set_plot_limits(ax2, cfg, '2')

    if 'xScale' in cfg['request']:
        plt.xscale(cfg['request']['xScale'])
    if 'yScale' in cfg['request']:
        plt.yscale(cfg['request']['yScale'])


    plt.grid()
    ea = None
    if lgd is not None:
        ea = (lgd,)
    plt.savefig(sys.argv[2], bbox_extra_artists=ea, bbox_inches='tight', dpi=200)
    plt.close('all')

cfg = {}
with open(sys.argv[1]) as json_file:
    cfg = json.load(json_file)

if cfg['request']['type'] == 'line':
    line_plot(cfg)

if cfg['request']['type'] == 'line-err':
    line_plot_err(cfg)
#!/bin/bash
mkdir -p ~/.debug
rm -rf ~/.debug/*
tar xvf data/testruns/$1/performanceprofiles/perf_$2.data.tar.bz2 -C ~/.debug
cd FlameGraph
rm -rf $2.perf-folded
./stackcollapse-perf.pl $PWD/../data/testruns/$1/performanceprofiles/perf_$2.script > $PWD/../data/testruns/$1/performanceprofiles/$2.perf-folded
./flamegraph.pl $PWD/../data/testruns/$1/performanceprofiles/$2.perf-folded > $PWD/../data/testruns/$1/plots/perf_flamegraph_2_$2.svg

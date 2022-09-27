import React from 'react';

const Dashboard = React.lazy(() => import('./pages/Dashboard'));
const TestRuns = React.lazy(() => import('./pages/TestRuns'));
const TestRun = React.lazy(() => import('./pages/TestRun/TestRun'));
const TestRunMatrix = React.lazy(() => import('./pages/TestRunMatrix'));
const ScheduleTestRun = React.lazy(() => import('./pages/ScheduleTestRun'));
const Config = React.lazy(() => import('./pages/Config'));
const Sweeps = React.lazy(() => import('./pages/Sweeps'));
const SweepPlot = React.lazy(() => import('./pages/SweepPlot'));
const Report = React.lazy(() => import('./pages/Report'));

const routes = [
  { path: '/', exact: true, name: 'Home' },
  { path: '/dashboard', name: 'Dashboard', component: Dashboard },
  { path: '/testruns/schedule', exact: true,  name: 'Schedule New Test Run', component: ScheduleTestRun },
  { path: '/testruns/sweepMatrix/:sweepID', exact: true,  name: 'Test Run Sweep Matrix', component: TestRunMatrix },
  { path: '/testruns/reschedule/:testRunID', exact: true,  name: 'Schedule New Test Run', component: ScheduleTestRun },
  { path: '/sweeps',  name: 'Sweeps', component: Sweeps },
  { path: '/sweepPlot/:sweepID',  name: 'Sweep Plot', component: SweepPlot },
  { path: '/testruns/:state/:spec',  name: 'Test Runs', component: TestRuns },
  { path: '/testruns/:state',  name: 'Test Runs', component: TestRuns },
  { path: '/testrun/:testRunID',  name: 'Test Run Details', component: TestRun },
  { path: '/config', exact: true,  name: 'Config', component: Config },
  { path: '/report', exact: true,  name: 'Report', component: Report }
];

export default routes;

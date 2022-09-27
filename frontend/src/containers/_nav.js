const _nav =  [
  {
    _tag: 'CSidebarNavItem',
    name: 'Dashboard',
    to: '/dashboard',
  },
  {
    _tag: 'CSidebarNavTitle',
    _children: ['Test Runs']
  },
  {
    _tag: 'CSidebarNavItem',
    name: 'Start new test run',
    to: '/testruns/schedule',
  },
  {
    _tag: 'CSidebarNavItem',
    name: 'Running test runs',
    to: '/testruns/running',
  },
  {
    _tag: 'CSidebarNavItem',
    name: 'Queued test runs',
    to: '/testruns/queued',
  },
  {
    _tag: 'CSidebarNavItem',
    name: 'Completed test runs',
    to: '/testruns/completed',
  },
  {
    _tag: 'CSidebarNavItem',
    name: 'Pending Peak Observation',
    to: '/testruns/pendingPeakObservation',
  },
  {
    _tag: 'CSidebarNavItem',
    name: 'Failed test runs',
    to: '/testruns/failed',
  },
  {
    _tag: 'CSidebarNavTitle',
    _children: ['Results']
  },
  {
    _tag: 'CSidebarNavItem',
    name: 'Sweeps',
    to: '/sweeps',
  },
  {
    _tag: 'CSidebarNavTitle',
    _children: ['System']
  },
  {
    _tag: 'CSidebarNavItem',
    name: 'Config',
    to: '/config',
  },
  {
    _tag: 'CSidebarNavDivider',
    className: 'm-2'
  }
]

export default _nav

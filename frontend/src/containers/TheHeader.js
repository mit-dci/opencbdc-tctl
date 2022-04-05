import React from 'react'
import { useSelector, useDispatch } from 'react-redux'
import {
  CHeader,
  CToggler,
  CHeaderBrand,
  CHeaderNav,
  CHeaderNavItem,
  CHeaderNavLink,
  CSubheader,
  CBreadcrumbRouter,
  CLink
} from '@coreui/react'
import CIcon from '@coreui/icons-react'

// routes config
import routes from '../routes'

const TheHeader = (props) => {
  const me = useSelector(state => state.users?.me);
  const maintenanceMode = useSelector(state => state.system?.maintenanceMode);

  return (
    <CHeader>
      <CHeaderNav className="d-md-down-none mr-auto">
      <CBreadcrumbRouter 
          className="border-0 c-subheader-nav m-0 px-0 px-md-3" 
          routes={routes} 
        />
      </CHeaderNav>
      {maintenanceMode === true && <CHeaderNav className="px-3" style={{color:'red', fontWeight:'bold'}}>
        System is in maintenance mode - test runs will be queued
      </CHeaderNav>}
      <CHeaderNav className="px-3">
        Logged in as:&nbsp;<b>{me?.name} ({me?.org})</b>
      </CHeaderNav>
    </CHeader>
  )
}

export default TheHeader

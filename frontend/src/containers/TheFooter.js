import React from 'react'
import { CFooter } from '@coreui/react'
import { useSelector } from 'react-redux'

const TheFooter = (props) => {

  const version = useSelector(state => state.system?.version);

  return (
    <CFooter fixed={false}>
      <div>
        <a href="https://coreui.io" target="_blank" rel="noopener noreferrer">CoreUI</a>
        <span className="ml-1">&copy; 2020 creativeLabs.</span>
        - 
        OpenCBDC Test Controller v{version}
      </div>
      <div className="mfs-auto">
        <span className="mr-1">Powered by</span>
        <a href="https://coreui.io/react" target="_blank" rel="noopener noreferrer">CoreUI for React</a>
      </div>
    </CFooter>
  )
}

export default React.memo(TheFooter)

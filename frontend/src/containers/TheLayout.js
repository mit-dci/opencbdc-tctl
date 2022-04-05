import React from 'react'
import {
  TheContent,
  TheSidebar,
  TheFooter,
  TheHeader
} from './index'
import { ToastContainer } from 'react-toastify';
import "react-toastify/dist/ReactToastify.css";

const TheLayout = (props) => {
  return (
    <div className="c-app c-default-layout">
      <TheSidebar/>
      <div className="c-wrapper">
        <TheHeader  {...props}/>
        <div className="c-body">
          <TheContent {...props}  />
        </div>
        <TheFooter {...props} />
      </div>
      <ToastContainer autoClose={3000} pauseOnFocusLoss={false} newestOnTop={true} />
    </div>
  )
}

export default TheLayout

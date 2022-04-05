import {CCard, CCardHeader, CCardBody} from "@coreui/react";

const SimpleCard = (props) => <CCard>
<CCardHeader>
  {props.center ? <center><b>{props.title}</b></center> : <b>{props.title}</b>}
</CCardHeader>
<CCardBody>
    {props.center ? <center>{props.children}</center> : props.children}
</CCardBody>
</CCard>

export default SimpleCard;


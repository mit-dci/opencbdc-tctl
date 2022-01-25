import React, { useState } from "react";
import {
  CCard,
  CCardBody,
  CCardHeader,
  CCol,
  CRow,
  CButton,
  CInput,
  CTextarea
} from "@coreui/react";
import {generateReport} from "../state/slices/testruns"
import { useDispatch } from "react-redux";

const Report = () => {
  const [reportDefinition, setReportDefinition] = useState("");
  const [reportTitle, setReportTitle] = useState("");

  const dispatch = useDispatch();

  return (
    <CRow>
      <CCol xl={6}>
        <CCard>
          <CCardHeader>Report definition</CCardHeader>
          <CCardBody>
            <CRow style={{marginBottom:'10px'}}>
              <CCol xs={3}>Title:</CCol>
              <CCol xs={9}><CInput type="text" value={reportTitle} onChange={(e) => {
                setReportTitle(e.target.value);
              }}></CInput></CCol>
            </CRow>
            <CRow style={{marginBottom:'10px'}}>
              <CCol><CTextarea rows={10} value={reportDefinition} onChange={(e) => {
                setReportDefinition(e.target.value);
              }}></CTextarea></CCol>
            </CRow>
            <CRow style={{marginBottom:'10px'}}>
              <CCol xs={12} style={{textAlign:'right'}}>
                <CButton color="primary" onClick={(e) => { dispatch(generateReport({title:reportTitle, definition:reportDefinition})) }}>
                  Generate
                </CButton>
              </CCol>
            </CRow>
          </CCardBody>
        </CCard>
      </CCol>
    </CRow>
  );
};

export default Report;

import React from "react";
import {
  CCol,
  CRow,
  CCard,
  CCardHeader,
  CCardBody,
} from "@coreui/react";

const TestRunParameters = (props) => {
  return (
    <CCard>
      <CCardHeader>
        <b>Test Run Parameters</b>
      </CCardHeader>
      <CCardBody>
        <CRow>
          {props.testRunFields.map((f) => {
            return (
              <>
                <CCol xs={2}>{f.title}:</CCol>
                <CCol xs={f.type === "commit" ? 10 : 4}>
                  {["int", "float", "loglevel"].indexOf(f.type) !== -1 && (
                    <b>{props.testRun[f.name]}</b>
                  )}
                  {f.type === "arch" && <b>{props.selectedArchitecture?.name}</b>}
                  {f.type === "bool" && (
                    <b>{props.testRun[f.name] === true ? "Yes" : "No"}</b>
                  )}
                  {f.type === "commit" && (
                    <b>
                      Commit {props.commit?.commit.substr(0, 8)}: {props.commit?.subject}
                    </b>
                  )}
                </CCol>
                {f.type === "commit" && <CCol xs={12}>&nbsp;</CCol>}
              </>
            );
          })}
        </CRow>
      </CCardBody>
    </CCard>
  );
};

export default TestRunParameters;
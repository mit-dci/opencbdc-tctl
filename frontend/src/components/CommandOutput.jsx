import React, { useEffect, useState } from "react";
import "./CommandOutput.css";
import AutoScrollingTextarea from "./AutoScrollingTextArea";
import {
  CButton,
  CCol,
  CRow,
  CCard,
  CCardHeader,
  CCardBody,
  CButtonGroup,
} from "@coreui/react";
import { useDispatch, useSelector } from "react-redux";
import { downloadCommandLog } from '../state/slices/testruns';
import client from '../state/apiclient';

const CommandOutput = (props) => {
  const dispatch = useDispatch();
  const [view, setView] = useState("out");
  useEffect(() => {
    if (props.command?.id) {
      dispatch(downloadCommandLog(props.command.id, view));
    }
  }, [props.command.id, view]);
  const log = useSelector(state => (state.testruns.activeCommandLog.id === props.command.id && state.testruns.activeCommandLog.view === view) ? state.testruns.activeCommandLog.log : '')

  return (
    <CRow>
      {props.command?.id && <CCol>
        <CCard>
          <CCardHeader>
            Command Output for command {props.command.id}
          </CCardHeader>
          <CCardBody>
            <CRow>
              <CCol xs={3}>Command:</CCol>
              <CCol xs={9}>
                <b>{props.command.command}</b>
              </CCol>
            </CRow>
            <CRow>
              <CCol xs={3}>Parameters:</CCol>
              <CCol xs={9}>
                [{props.command.params && props.command.params.map(p => <><b>{p}</b>{', '}</>)}]
                </CCol>
            </CRow>
            <CRow>
              <CCol xs={3}>Environment:</CCol>
              <CCol xs={9}>
                [{props.command.env && props.command.env.map(p => <><b>{p}</b>{', '}</>)}]
                </CCol>
            </CRow>
            <CRow>
              <CCol xs={3}>Agent:</CCol>
              <CCol xs={9}>
                <b>{props.command.agent}</b>
              </CCol>
            </CRow>
            <CRow>
              <CCol xs={3}>Status:</CCol>
              <CCol xs={9}>
                <b>{props.command.status}</b>
              </CCol>
            </CRow>
          </CCardBody>
        </CCard>
        <CCard>
          <CCardHeader>
            <CRow>
              <CCol xs={7}>
                <CButtonGroup>

                  <CButton variant="outline"
                    color="secondary"
                    outline
                    active={view === "out"}
                    onClick={(e) => setView("out")}
                  >
                    Stdout
                </CButton>
                  <CButton variant="outline"
                    color="secondary"
                    outline
                    active={view === "err"}
                    onClick={(e) => setView("err")}
                  >
                    Stderr
                </CButton>
                </CButtonGroup>
              </CCol>
              <CCol xs={5}>
                <CButton
                style={{float:'right'}}
                  color="secondary"
                  outline
                  onClick={(e) => {
                    window.open(`${client.apiUrl}commands/${props.command.id}/output/${view}/full`)
                  }}
                >
                  Full
                </CButton>
              </CCol>
            </CRow>
          </CCardHeader>
          <CCardBody>
            <AutoScrollingTextarea
              className="log"
              value={log}
            ></AutoScrollingTextarea>
          </CCardBody>
        </CCard>
      </CCol>}
    </CRow>
  );
}

export default CommandOutput;

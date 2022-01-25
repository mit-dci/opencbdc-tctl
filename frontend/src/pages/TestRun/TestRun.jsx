import React, { useState, useEffect } from "react";
import client from '../../state/apiclient';
import * as moment from "moment";
import * as numeral from "numeral";
import Moment from "react-moment";
import AutoScrollingTextarea from "../../components/AutoScrollingTextArea";
import TestRunParameters from './TestRunParameters';
import PerformanceData from './PerformanceData';
import {
  CButton,
  CCol,
  CRow,
  CCard,
  CCardHeader,
  CCardBody,
  CDataTable,
  CButtonGroup,
} from "@coreui/react";
import { useHistory, useParams } from "react-router-dom";
import CommandOutput from "../../components/CommandOutput";
import "./TestRun.css";
import TestResult from "../../components/TestResult";
import User from "../../components/User";
import { useDispatch, useSelector } from 'react-redux';
import {loadTestRunDetails, retrySpawning, terminateTestRun, subscribeTestRunLog, unsubscribeTestRunLog, selectTestRunRunningCommands} from '../../state/slices/testruns';


const TestRun = (props) => {
  const history = useHistory();
  const params = useParams();
  const dispatch = useDispatch();
  const [detailCommand, setDetailCommand] = useState({ type: "", id: "" });
  const [view, setView] = useState("details");


  

  const initialStateLoaded = useSelector(state => state.system.initialStateLoaded)
  const testRun = useSelector(state => state.testruns.testruns.find((tr) => tr.id === params.testRunID));
  const testRunFields = useSelector(state => state.testruns.testrunFields);
  const webSocketConn = useSelector(state => state.system.webSocketConn);
  useEffect(() => {
    dispatch(subscribeTestRunLog(params.testRunID));
    return () => { dispatch(unsubscribeTestRunLog()); }
  }, [params.testRunID, webSocketConn]);

  const selectedArchitecture = useSelector(state => state.architectures.architectures.find((a) => a.id === (testRun?.architectureID || 'default')));
  const launchTemplates = useSelector(state => state.agents.launchTemplates);
  const testRunLog = useSelector(state => {
     return (view === "log" ? state.testruns.testrunLogs.find((trl) => (trl.id === params.testRunID)) : "")
  });
  const showCommand = (cmd) => {
    const lastSlash = cmd.lastIndexOf("/") + 1;
    return lastSlash === -1 ? cmd : cmd.substring(lastSlash);
  };
  const commit = useSelector((state) => state.commits?.commits.find(c => c.commit == testRun.commitHash));

  if (!initialStateLoaded) {
    return (
      <CRow>
        <CCol>
          <h1>Waiting for initial system state</h1>
        </CCol>
      </CRow>
    );
  }


  if (!testRun || testRun.id !== params.testRunID) {
    return (
      <CRow>
        <CCol>
          <h1>Not found</h1>
        </CCol>
      </CRow>
    );
  }

  if (testRun.detailsAvailable !== true) {
    if (testRun.detailsLoading !== true) {
      dispatch(loadTestRunDetails(testRun.id));
    }
    return (
      <CRow>
        <CCol>
          <h1>Loading...</h1>
        </CCol>
      </CRow>
    );
  }

  return (
    <CRow>
      <CCol xs={12} style={{textAlign:'center', paddingBottom: '10px'}}>
        <CButtonGroup>
          <CButton color="primary" variant="outline" active={view === "details"} onClick={(e) => { setView("details") }}>Details &amp; Parameters</CButton>
          {testRun.status === "Completed" && <CButton color="primary" variant="outline" active={view === "results"} onClick={(e) => { setView("results") }}>Results</CButton>}
          {(testRun.status === "Completed" || testRun.status === "Failed") && <CButton color="primary" variant="outline" active={view === "performance"} onClick={(e) => { setView("performance") }}>Performance Data</CButton>}
          <CButton color="primary" variant="outline" active={view === "roles"} onClick={(e) => { setView("roles") }}>Roles</CButton>
          <CButton color="primary" variant="outline" active={view === "commands"} onClick={(e) => { setView("commands") }}>Commands</CButton>
          <CButton color="primary" variant="outline" active={view === "log"} onClick={(e) => { setView("log") }}>Log</CButton>
        </CButtonGroup>
      </CCol>
      {view === "results" &&
        <CCol xs={12}>
          <TestResult testRun={testRun} />
        </CCol>
      }
      {view === "performance" && <CCol xs={12}>
        <PerformanceData testRun={testRun} />
      </CCol>}
      {view === "roles" && <CCol xs={12}>
        <CCard>
          <CCardHeader>
            <b>Test Run Roles</b>
          </CCardHeader>
          <CCardBody>
            <CRow>
              {testRun.roles.map((r) => {
                var role = selectedArchitecture.roles.find(
                  (ar) => ar.role === r.role
                );
                var agents = testRun.testrunAgentData || [];
                var agent = agents.find((a) => a.agentID === r.agentID);
                var awsConfig = launchTemplates?.find(lt => lt.id === r.awsLaunchTemplateID)
                var config = "Unknown";

                if (agent) {
                  config = `${agent.systemInfo?.numCPU} CPU / ${numeral(
                    agent.systemInfo?.memAvailable / 1024 / 1024
                  ).format("#0.0")} GB RAM / ${numeral(
                    agent.systemInfo?.diskAvailable / 1024 / 1024
                  ).format("#0.0")} GB Disk`;
                } else if (awsConfig) {
                  config = awsConfig.description;
                }

                if(r.failure) {
                  config += ` - Fail after ${r.failure.after}s`;
                }
                return (
                  <>
                    <CCol xs={4}>
                      <CRow>
                        <CCol xs={4}>
                          <b>{`${role.title} ${r.roleIdx + 1}`}:</b>
                        </CCol>
                        <CCol xs={8}>{config}</CCol>
                      </CRow>
                    </CCol>
                  </>
                );
              })}
            </CRow>
          </CCardBody>
        </CCard>
      </CCol>}
      {view === "details" &&
      <CCol xs={testRun.status === "Running" ? 6 : 12}>
        <CCard>
          <CCardHeader>
            <b>Test Run Details</b>
          </CCardHeader>
          <CCardBody>
            <CRow>
              <CCol xs={2}>ID:</CCol>
              <CCol xs={4}>
                <b>{testRun.id}</b>
              </CCol>
              <CCol xs={2}>Created:</CCol>
              <CCol xs={4}>
                <b>
                  <Moment format="L">{testRun.created}</Moment>{" "}
                  <Moment format="LT">{testRun.created}</Moment>
                </b>
              </CCol>
            </CRow>
            <CRow>
              <CCol xs={2}>Status:</CCol>
              <CCol xs={4}>
                <b>{testRun.status}</b>
              </CCol>
              <CCol xs={2}>Started:</CCol>
              <CCol xs={4}>
                {!testRun.started.startsWith("0001") && (
                  <b>
                    <Moment format="L">{testRun.started}</Moment>{" "}
                    <Moment format="LT">{testRun.started}</Moment>
                  </b>
                )}
                &nbsp;
              </CCol>
            </CRow>
            <CRow>
              <CCol xs={2}>Details:</CCol>
              <CCol xs={4}>
                <b>{testRun.details}</b>
                {testRun.details &&
                  testRun.details.indexOf("AWS agents to come online") > -1 && (
                    <CButton
                      onClick={(e) => {
                        dispatch(retrySpawning(testRun.id));
                      }}
                      size="sm"
                      className="btn-pill"
                      color="primary"
                    >
                      Retry spawning
                    </CButton>
                  )}
              </CCol>
              <CCol xs={2}>Completed:</CCol>
              <CCol xs={4}>
                {!testRun.completed.startsWith("0001") && (
                  <b>
                    <Moment format="L">{testRun.completed}</Moment>{" "}
                    <Moment format="LT">{testRun.completed}</Moment>
                  </b>
                )}
                &nbsp;
              </CCol>{" "}
            </CRow>
            <CRow>
              <CCol xs={2}>Created by:</CCol>
              <CCol xs={4}>
                <b>
                  <User thumbPrint={testRun.createdByuserThumbprint} />
                </b>
              </CCol>
              {testRun.status === "Completed" && (
                <CCol xs={{ size: 2, offset: 2 }}>
                  <CButton
                    block
                    color="primary"
                    onClick={(e) => {
                      window.open(`${client.apiUrl}testruns/${testRun.id}/outputs`);
                    }}
                  >
                    Download raw output
                  </CButton>
                </CCol>
              )}
            </CRow>
          </CCardBody>
        </CCard>
      </CCol>}
      {view === "details" && testRun.status === "Running" && (
        <CCol xs={6}>
          <CCard>
            <CCardHeader>
              <b>Actions</b>
            </CCardHeader>
            <CCardBody>
              <CRow>
                <CButton
                  block
                  xs={{ size: 4, offset: 4 }}
                  color="primary"
                  onClick={(e) => {
                    dispatch(terminateTestRun(params.testRunID));
                  }}
                >
                  Terminate
                </CButton>
              </CRow>
            </CCardBody>
          </CCard>
        </CCol>
      )}
      {view === "details" &&  <CCol xs={12}>
           <TestRunParameters testRunFields={testRunFields} testRun={testRun} commit={commit} selectedArchitecture={selectedArchitecture} />
      </CCol>}
      {view === "commands" && <><CCol xs={6}>
        <CCard>
          <CCardHeader>
            <b>Completed commands ({testRun.executedCommands.length})</b>
          </CCardHeader>
          <CCardBody
            style={{
              overflowY: "auto",
              height: testRun.status === "Running" ? "300px" : "600px",
            }}
          >
            <CDataTable
              items={testRun.executedCommands.map((ec) => {
                return {
                  Agent: ec.agentID,
                  Command: showCommand(ec.description),
                  Status: `Completed ${moment(ec.completed).format("LTS")} (${
                    ec.returnCode
                  })`,
                  Actions: { ec: ec },
                };
              })}
              scopedSlots={{
                Actions: (ec) => {
                  return (
                    <td>
                      <CButton
                        color="primary"
                        onClick={(e) => {
                          setDetailCommand({
                            agent: ec.Actions.ec.agentID,
                            status: ec.Status,
                            command: ec.Actions.ec.description,
                            env:  ec.Actions.ec.env,
                            params:  ec.Actions.ec.params,
                            id: ec.Actions.ec.commandID,
                            type: "executed",
                          });
                        }}
                      >
                        ...
                      </CButton>
                    </td>
                  );
                },
              }}
            />
          </CCardBody>
        </CCard>
      </CCol>
      <CCol xs={6}>
        <CommandOutput
          testRun={testRun}
          command={detailCommand}
        />
      </CCol>
      </>
      }
      {view === "log" &&
        <CCol xs={12}>
          <CCard>
            <CCardHeader>
              <CRow>
                <CCol xs={6}><b>Test Run Log</b></CCol>
                <CCol xs={6}><CButton
                style={{float:'right'}}
                  color="secondary"
                  outline
                  onClick={(e) => {
                    window.open(`${client.apiUrl}testruns/${testRun.id}/log`)
                  }}
                >
                  Full
                </CButton></CCol>
                </CRow>
            </CCardHeader>
            <CCardBody>
              <AutoScrollingTextarea
              className="log"
              value={testRunLog?.log}
              ></AutoScrollingTextarea>
            </CCardBody>
          </CCard>
        </CCol>
      }
    </CRow>
  );
};

export default TestRun;

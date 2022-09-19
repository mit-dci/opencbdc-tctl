import React from "react";
import * as numeral from "numeral";
import * as moment from "moment";
import {
  CButton,
  CCard,
  CDataTable,
  CCardBody,
  CCardFooter,
  CCardHeader,
  CCol,
  CCollapse,
  CDropdownItem,
  CDropdownMenu,
  CDropdownToggle,
  CFade,
  CForm,
  CFormGroup,
  CFormText,
  CValidFeedback,
  CInvalidFeedback,
  CTextarea,
  CInput,
  CInputFile,
  CInputCheckbox,
  CInputRadio,
  CInputGroup,
  CInputGroupAppend,
  CInputGroupPrepend,
  CDropdown,
  CInputGroupText,
  CLabel,
  CSelect,
  CRow,
  CModal,
  CModalHeader,
  CModalBody,
  CModalFooter,
  CSwitch,
} from "@coreui/react";

import Selector from "../components/Selector";

import { useHistory, withRouter } from "react-router";
import { useDispatch, useSelector } from "react-redux";
import {
  selectScheduledRunRoles,
  selectScheduledRunSweepRoles,
  setScheduledRunProperty,
  validateAndScheduleTestRun,
  estimateTestRun,
  applySweepRunRoleConfig,
  applyScheduledRunRoleConfig,
  applyScheduledRunRoleFailure,
  setScheduledRunRoleFail,
  deleteScheduledRunRole,
  setScheduledRunRoleAgent,
  deleteScheduledRunSweepRole,
  setScheduledRunSweepRoleAgent,
} from "../state/slices/testruns";
import { useState } from "react";

const ScheduledRunRole = (props) => <CCol style={{border: '1px solid #a0a0a0', borderRadius: '2px', backgroundColor: '#f0f0f0', marginBottom: '2px'}}>
    <CRow>
      <CCol xs={6}><b>{`${props.role.Role} ${props.role.Index}`}</b></CCol>
      <CCol xs={2}><b>{`${props.role.Agent.launchTemplate.vCPU}`}</b></CCol>
      <CCol xs={4} style={{textAlign:'right'}}><b>{`${props.role.Agent.failure ? `Fail after ${props.role.Agent.failure.after}s` : 'No Fail'}`}</b></CCol>
    </CRow>
  </CCol>;

const ScheduledRunProperty = (props) => {
  const dispatch = useDispatch();
  const architectures = useSelector(
    (state) => state.architectures?.architectures
  );
  const commits = useSelector((state) => state.commits?.commits);

  let titleSize = 8;
  if(props.xs == 6) {
    titleSize = 4;
  }
  if(props.xs == 12) {
    titleSize = 2;
  }


  return (
    <CCol xs={props.xs}>
      <CFormGroup row>
        <CCol sm={titleSize}>
          <CLabel htmlFor={props.id}>{props.title}:</CLabel>
        </CCol>
        <CCol sm={12-titleSize}>
          {props.type === "bool" && (
            <CInputCheckbox
              style={{ marginLeft: 0 }}
              id={props.id}
              checked={props.value === true}
              onChange={(e) => {
                let obj = {};
                obj[props.id] = e.target.checked;
                dispatch(setScheduledRunProperty(obj));
              }}
            ></CInputCheckbox>
          )}
          {props.type === "txtype" && (
            <Selector
            values={["transfer", "erc20"]}
            valueFunc={(c) => c}
            displayFunc={(c) => c}
            value={props.value}
            id={props.id}
            onChange={(e) => {
              var obj = {};
              obj[props.id] = e.target.value;
              dispatch(setScheduledRunProperty(obj));
            }}
          />
          )}
          {props.type === "loglevel" && (
            <Selector
            values={["FATAL", "ERROR", "WARN", "INFO", "DEBUG", "TRACE"]}
            valueFunc={(c) => c}
            displayFunc={(c) => c}
            value={props.value}
            id={props.id}
            onChange={(e) => {
              var obj = {};
              obj[props.id] = e.target.value;
              dispatch(setScheduledRunProperty(obj));
            }}
          />
          )}
          {props.type === "arch" && (
            <Selector
            values={architectures}
            valueFunc={(a) => a.id}
            displayFunc={(a) => a.name}
            value={props.value}
            id={props.id}
            onChange={(e) => {
              var obj = {};
              obj[props.id] = e.target.value;
              dispatch(setScheduledRunProperty(obj));
            }}
          />
          )}
           {props.type === "commit" && (
            <Selector
            values={commits}
            valueFunc={(c) => c.commit}
            displayFunc={(c) => `${c.commit.substring(0,6)} - ${moment(c.committed).fromNow()}${c.author.name ? ' - ' : ''}${c.author.name} - ${c.subject}`}
            value={props.value}
            id={props.id}
            onChange={(e) => {
              var obj = {};
              obj[props.id] = e.target.value;
              dispatch(setScheduledRunProperty(obj));
            }}
          />
          )}
          {["int","float"].indexOf(props.type) > -1 && (
            <CInput
              type={props.type}
              id={props.id}
              checked={props.value === true}
              value={props.value}
              onChange={(e) => {
                let obj = {};
                obj[props.id] = e.target.value;
                if (props.type === "int") {
                  let val = parseInt(e.target.value);
                  if (Number.isNaN(val)) val = 0;
                  obj[props.id] = val;
                }
                if (props.type === "float") {
                  let val = parseFloat(e.target.value);
                  if (Number.isNaN(val)) {
                    val = 0;
                  } else if (e.target.value.endsWith(".")) {
                    val = val.toString() + ".";
                  }
                  obj[props.id] = val;
                }
                dispatch(setScheduledRunProperty(obj));
              }}
            ></CInput>
          )}
          {}
        </CCol>
      </CFormGroup>
    </CCol>
  );
};


const ScheduleTestRun = (props) => {
  const initialStateLoaded = useSelector(state => state.system.initialStateLoaded)

  const scheduledRun = useSelector((state) => state.testruns?.scheduledTestRun);
  
  const selectedArchitecture = useSelector((state) =>
    state.architectures?.architectures?.find(
      (a) => a.id === (scheduledRun?.architectureID || "default")
    )
  );
  const availableRoles = selectedArchitecture?.roles || [];
  const dispatch = useDispatch();
  const launchTemplates = useSelector((state) => state.agents.launchTemplates);
  const scheduledRunRoles = useSelector(selectScheduledRunRoles);
  const scheduledRunSweepRoles = useSelector(selectScheduledRunSweepRoles);
  const history = useHistory();
  const testRunFields = useSelector(state => state.testruns.testrunFields);
 
  const [configRole, setConfigRole] = useState("");
  const [configRoleCount, setConfigRoleCount] = useState("1");
  const [configRoleRegion, setConfigRoleRegion] = useState("all");
  const [configRoleInstanceType, setConfigRoleInstanceType] = useState("");

  const [sweepConfigRole, setSweepConfigRole] = useState("");
  const [sweepConfigRoleCount, setSweepConfigRoleCount] = useState("1");
  const [sweepConfigRoleRegion, setSweepConfigRoleRegion] = useState("all");
  const [sweepConfigRoleInstanceType, setSweepConfigRoleInstanceType] = useState("");


  const [failWhat, setFailWhat] = useState("");
  const [failAfter, setFailAfter] = useState("120");

  const [confirmModal, setConfirmModal] = useState(false);
  const [testRunEstimate, setTestRunEstimate] = useState({});

  const toggleConfirm = ()=>{
    setConfirmModal(!confirmModal);
  }

  if (!initialStateLoaded) {
    return (
      <CRow>
        <CCol>
          <h1>Waiting for initial system state</h1>
        </CCol>
      </CRow>
    );
  }
  let instanceTypes = [];
  let regions = [];
  for (let launchTemplate of launchTemplates) {
    if (
      !instanceTypes.find(
        (it) => it.instanceType === launchTemplate.instanceType
      )
    ) {
      instanceTypes.push(launchTemplate);
    }
    if (!regions.find((r) => r === launchTemplate.region)) {
      regions.push(launchTemplate.region);
    }
  }



  let agentChoices = [];
  if (launchTemplates && launchTemplates.length > 0) {
    agentChoices = launchTemplates.map((a) => {
      return {
        value: `AWS-${a.id}`,
        desc: `Boot new AWS agent (${a.description})`,
      };
    });
  }

  let regionRoles = [];
  for(let region of regions) {
    for(let role of availableRoles) {
      regionRoles.push({region, role})
    }
  }

  const prices = {
    "c5n.large" : 0.12,
    "c5n.2xlarge" : 0.45,
    "c5n.9xlarge" : 1.96,
    "c5n.metal" : 3.9,
  }

  let estimatedInstanceHours = [];
  let estimatedCharges = 0;

  if(testRunEstimate.instanceHours) {
    for(var prop in testRunEstimate.instanceHours) {
      if(testRunEstimate.instanceHours.hasOwnProperty(prop)) {
        estimatedInstanceHours.push({
          type : prop,
          hours :  testRunEstimate.instanceHours[prop],
        })
        let price = 0;
        if(prices[prop]) {
          price = prices[prop];
        }
        estimatedCharges += testRunEstimate.instanceHours[prop] * price;
      }
    }
  }


  return (
    <>
      <CRow>
        <CCol>
          <CForm>
            <CRow>
              <CCol xs={12}>
                <CCard>
                  <CCardHeader><b>Parameters</b></CCardHeader>
                  <CCardBody>
                    <CRow>
                      {testRunFields.map(f => <ScheduledRunProperty
                        xs={(f.type === 'commit' ? 12 : (f.type === 'arch' ? 6 : 3))}
                        title={f.title}
                        type={f.type}
                        id={f.name}
                        value={scheduledRun[f.name]}
                      />)}
                    </CRow>
                  </CCardBody>
                </CCard>
              </CCol>
              <CCol xs={12}>
                <CCard>
                  <CCardHeader><b>Roles</b></CCardHeader>
                  <CCardBody>
                    {availableRoles && availableRoles.length > 0 && (
                      <CRow
                        style={{
                          paddingBottom: "10px",
                          borderBottom: "1px solid #e0e0e0",
                        }}
                      >
                        <CCol xs={12}>
                          <CRow>
                            <CCol xs={2}>
                              <b>Configure Role:</b>
                            </CCol>
                            <CCol xs={1}>
                              <b>Count:</b>
                            </CCol>
                            <CCol xs={2}>
                              <b>Region:</b>
                            </CCol>
                            <CCol xs={4}>
                              <b>Instance type:</b>
                            </CCol>
                          </CRow>
                          <CRow>
                            <CCol xs={2}>
                              <Selector
                                values={availableRoles}
                                valueFunc={(a) => a.role}
                                displayFunc={(a) => a.title}
                                value={configRole}
                                onChange={(e) => {
                                  setConfigRole(e.target.value);
                                }}
                              />
                            </CCol>
                            <CCol xs={1}>
                              <CInput
                                type="text"
                                value={configRoleCount}
                                onChange={(e) => {
                                  setConfigRoleCount(e.target.value);
                                }}
                              />
                            </CCol>
                            <CCol xs={2}>
                              <Selector
                                values={[...regions, "all"]}
                                valueFunc={(a) => a}
                                displayFunc={(a) => a}
                                value={configRoleRegion}
                                onChange={(e) => {
                                  setConfigRoleRegion(e.target.value);
                                }}
                              />
                            </CCol>
                            <CCol xs={4}>
                              <Selector
                                values={instanceTypes}
                                valueFunc={(a) => a.instanceType}
                                displayFunc={(a) =>
                                  `${a.instanceType} (${a.vCPU} / ${a.ram} / ${a.bandwidth})`
                                }
                                value={configRoleInstanceType}
                                onChange={(e) => {
                                  setConfigRoleInstanceType(e.target.value);
                                }}
                              />
                            </CCol>
                            <CCol xs={3}>
                              <CButton
                                shape="square"
                                color="primary"
                                onClick={(e) => {
                                  let roleLaunchTemplates = launchTemplates.filter(lt => {
                                    return (
                                      (configRoleRegion === 'all' || lt.region === configRoleRegion) &&
                                      (lt.instanceType === configRoleInstanceType)
                                    )
                                  });



                                  dispatch(
                                    applyScheduledRunRoleConfig({
                                      role: configRole,
                                      agentChoices: roleLaunchTemplates,
                                      count: configRoleCount,
                                    })
                                  );
                                }}
                              >
                                Apply
                              </CButton>
                            </CCol>
                          </CRow>
                        </CCol>
                      </CRow>
                    )}
                    {launchTemplates && launchTemplates.length > 0 &&
                    <CRow style={{paddingTop: '30px'}}>
                    {regions.map(r => <CCol style={{width:`${100/regions.length}%`}}>
                      <CCard>
                        <CCardHeader><b>{r}</b></CCardHeader>
                        <CCardBody>
                          {scheduledRunRoles.filter(srr => srr.Agent.launchTemplate?.region === r).map(sr => <CRow>
                            <ScheduledRunRole role={sr} />
                          </CRow>)}
                        </CCardBody>
                      </CCard>
                    </CCol>)}
                    </CRow>}
                    {(!launchTemplates || launchTemplates.length === 0) && <CDataTable
                      items={scheduledRunRoles}
                      scopedSlots={{
                        Delete: (r) => {
                          return (
                            <td>
                              <CButton
                                size="sm"
                                className="btn-pill"
                                style={{ marginRight: "3px" }}
                                color="danger"
                                key={r.role}
                                onClick={(e) => {
                                  dispatch(
                                    deleteScheduledRunRole(r.Delete.role)
                                  );
                                }}
                              >
                                Del
                              </CButton>
                            </td>
                          );
                        },
                        Fail: (r) => {
                          return (
                            <td style={{ textAlign: "center" }}>
                              <CInputCheckbox
                                checked={r.Fail.fail}
                                onChange={(e) => {
                                  dispatch(
                                    setScheduledRunRoleFail(
                                      r.Fail.role,
                                      r.Fail.roleIdx,
                                      e.target.checked
                                    )
                                  );
                                }}
                              />
                            </td>
                          );
                        },
                        Agent: (r) => {
                          return (
                            <td>
                              <Selector
                                values={agentChoices}
                                valueFunc={(a) => a.value}
                                displayFunc={(a) => a.desc}
                                value={r.Agent.agentChoiceId}
                                onChange={(e) => {
                                  dispatch(
                                    setScheduledRunRoleAgent(
                                      r.Agent.role,
                                      r.Agent.roleIdx,
                                      e.target.value
                                    )
                                  );
                                }}
                              />
                            </td>
                          );
                        },
                      }}
                    />}
                  </CCardBody>
                </CCard>
              </CCol>
            </CRow>
            <CRow>
              <CCol xs={12}>
                  <CCard>
                    <CCardHeader><b>Failures</b></CCardHeader>
                    <CCardBody>
                          <CRow>
                            <CCol xs={4}>
                              <b>Set failure for:</b>
                            </CCol>
                            <CCol xs={4}>
                              <b>Failure mode:</b>
                            </CCol>
                          </CRow>
                          <CRow>
                            <CCol xs={4}>
                              <Selector
                                values={[
                                  ...regions.map(r => {return{
                                    id: `REGION-${r}`,
                                    desc: `Entire region ${r}`,
                                  }}),
                                  ...regionRoles.map(r => {return{
                                    id: `REGIONROLE-${r.region}|${r.role.role}`,
                                    desc: `${r.role.title}s in region ${r.region}`,
                                  }}),
                                  ...scheduledRunRoles.map(r => {return{
                                    id: `ROLE-${r.Agent.role}|${r.Agent.roleIdx}`,
                                    desc: `Role ${r.Role} ${r.Index}`,
                                  }})
                                ]}
                                valueFunc={(a) => a.id}
                                displayFunc={(a) => a.desc}
                                value={failWhat}
                                onChange={(e) => {
                                  setFailWhat(e.target.value)
                                }}
                              />
                            </CCol>
                            <CCol xs={3}>
                              <CFormGroup row>
                                <CCol xs={3}><CInputRadio checked={failAfter > -1} onChange={(e) => setFailAfter(e.target.checked ? (failAfter > 0 ? failAfter : 120) : -1)} /> Fail {failAfter > -1 ? 'after:' : ''}</CCol>
                                {failAfter > -1 && <CCol xs={5}><CInput type="text" value={failAfter} onChange={(e) => setFailAfter(e.target.value)} /></CCol>}
                                {failAfter > -1 && <CCol xs={4}><CLabel>seconds</CLabel></CCol>}
                              </CFormGroup>
                            </CCol>
                            <CCol xs={3}>
                              <CFormGroup row>
                                <CCol xs={12}><CInputRadio checked={failAfter === -1} onChange={(e) => setFailAfter(e.target.checked ? -1 : 0)} /> Do not fail</CCol>
                              </CFormGroup>
                            </CCol>
                            <CCol xs={2}>
                              <CButton
                                shape="square"
                                color="primary"
                                onClick={(e) => {
                                  let what = [];
                                  if(failWhat.startsWith('ROLE-')) {
                                    let whatParts = failWhat.substring(5).split('|');
                                    what.push({
                                      role: whatParts[0],
                                      index: parseInt(whatParts[1]),
                                    });
                                  } else if(failWhat.startsWith('REGIONROLE-')) {
                                    let whatParts = failWhat.substring(11).split('|');
                                    what = scheduledRunRoles.filter(srr => (srr.Agent.launchTemplate?.region === whatParts[0] && srr.Agent.role === whatParts[1])).map(sr => {
                                      return {role:sr.Agent.role, index:sr.Agent.roleIdx}
                                    })
                                  } else if(failWhat.startsWith('REGION-')) {
                                    let failRegion = failWhat.substring(7)
                                    what = scheduledRunRoles.filter(srr => srr.Agent.launchTemplate?.region === failRegion).map(sr => {
                                      return {role:sr.Agent.role, index:sr.Agent.roleIdx}
                                    })
                                  }
                                  dispatch(
                                    applyScheduledRunRoleFailure({
                                      what: what,
                                      after: failAfter
                                    })
                                  );
                                }}
                              >
                                Apply
                              </CButton>
                            </CCol>
                          </CRow>
                    </CCardBody>
                  </CCard>
              </CCol>
            </CRow>
            <CRow>
              <CCol xs={12}>
                <CCard>
                  <CCardHeader>
                    <b>Parameter Sweep</b>
                  </CCardHeader>
                  <CCardBody>
                    <CRow>
                      <CCol xs={{ size: 3, offset: 1 }}>
                        <CRow>
                          <CInputRadio
                            id="sweep"
                            checked={scheduledRun.sweep === ""}
                            onChange={(e) =>
                              dispatch(setScheduledRunProperty({ sweep: "" }))
                            }
                          />{" "}
                          No sweep
                        </CRow>
                        <CRow>
                          <CInputRadio
                            id="sweep"
                            checked={scheduledRun.sweep === "parameter"}
                            onChange={(e) =>
                              dispatch(
                                setScheduledRunProperty({ sweep: "parameter" })
                              )
                            }
                          />{" "}
                          Parameter
                        </CRow>
                        <CRow>
                          <CInputRadio
                            id="sweep"
                            checked={scheduledRun.sweep === "time"}
                            onChange={(e) =>
                              dispatch(
                                setScheduledRunProperty({ sweep: "time" })
                              )
                            }
                          />{" "}
                          Time
                        </CRow>
                        <CRow>
                          <CInputRadio
                            id="sweep"
                            checked={scheduledRun.sweep === "peak"}
                            onChange={(e) =>
                              dispatch(
                                setScheduledRunProperty({ sweep: "peak" })
                              )
                            }
                          />{" "}
                          Peak Finding
                        </CRow>
                        {launchTemplates && launchTemplates.length > 0 && <CRow>
                          <CInputRadio
                            id="sweep"
                            checked={scheduledRun.sweep === "roles"}
                            onChange={(e) =>
                              dispatch(
                                setScheduledRunProperty({ sweep: "roles" })
                              )
                            }
                          />{" "}
                          Roles
                        </CRow>}
                        <CRow>
                          <CInputCheckbox
                            id="sweepOneAtATime"
                            checked={scheduledRun.sweepOneAtATime === true}
                            onChange={(e) =>
                              dispatch(
                                setScheduledRunProperty({ sweepOneAtATime: e.target.checked })
                              )
                            }
                          />{" "}
                          One at a time
                        </CRow>
                      </CCol>
                      {scheduledRun.sweep === "parameter" && (
                        <CCol xs={8}>
                          <CCard>
                            <CCardHeader>
                              <b>Parameter</b>
                            </CCardHeader>
                            <CCardBody>
                              <Selector
                                values={Object.keys(scheduledRun).filter(
                                  (key) =>
                                    !Number.isNaN(parseInt(scheduledRun[key]))
                                )}
                                valueFunc={(a) => a}
                                displayFunc={(a) => a}
                                value={scheduledRun.sweepParameterParam}
                                onChange={(e) => {
                                  dispatch(
                                    setScheduledRunProperty({
                                      sweepParameterParam: e.target.value,
                                      sweepParameterStart:
                                        scheduledRun[e.target.value],
                                      sweepParameterIncrement: 0,
                                      sweepParameterStop:
                                        scheduledRun[e.target.value],
                                    })
                                  );
                                }}
                              />
                            </CCardBody>
                          </CCard>
                          <CCard>
                            <CCardHeader>
                              <b>Range</b>
                            </CCardHeader>
                            <CCardBody>
                              <CRow>
                                <ScheduledRunProperty
                                  xs={12}
                                  title="Initial value"
                                  type="float"
                                  id="sweepParameterStart"
                                  value={scheduledRun.sweepParameterStart || 0}
                                />
                                <ScheduledRunProperty
                                  xs={12}
                                  title="Increment"
                                  type="float"
                                  id="sweepParameterIncrement"
                                  value={
                                    scheduledRun.sweepParameterIncrement || 0
                                  }
                                />
                                <ScheduledRunProperty
                                  xs={12}
                                  title="Final value"
                                  type="float"
                                  id="sweepParameterStop"
                                  value={scheduledRun.sweepParameterStop || 0}
                                />
                              </CRow>
                            </CCardBody>
                          </CCard>
                        </CCol>
                      )}
                      {scheduledRun.sweep === "time" && (
                        <CCol xs={8}>
                          <CCard>
                            <CCardHeader>
                              <b>Repetition</b>
                            </CCardHeader>
                            <CCardBody>
                              <CRow>
                                <ScheduledRunProperty
                                  xs={12}
                                  title="Number of total runs"
                                  type="int"
                                  id="sweepTimeRuns"
                                  value={scheduledRun.sweepTimeRuns || 5}
                                />
                              </CRow>
                              <CRow>
                                <ScheduledRunProperty
                                  xs={12}
                                  title="Number of minutes between runs"
                                  type="int"
                                  id="sweepTimeMinutes"
                                  value={scheduledRun.sweepTimeMinutes || 60}
                                />
                              </CRow>
                            </CCardBody>
                          </CCard>
                        </CCol>
                      )}
                      {scheduledRun.sweep === "roles" && (
                        <CCol xs={8}>
                          <CCard>
                            <CCardHeader>
                              <b>Repetition</b>
                            </CCardHeader>
                            <CCardBody>
                              <CRow>
                                <ScheduledRunProperty
                                  xs={12}
                                  title="Number of total runs"
                                  type="int"
                                  id="sweepRoleRuns"
                                  value={scheduledRun.sweepRoleRuns || 5}
                                />
                              </CRow>
                            </CCardBody>
                          </CCard>
                          <CCard>
                            <CCardHeader>
                              <b>Add these roles for each subsequent run:</b>
                            </CCardHeader>
                            <CCardBody>
                            <CRow>
                            <CCol xs={2}>
                              <Selector
                                values={availableRoles}
                                valueFunc={(a) => a.role}
                                displayFunc={(a) => a.title}
                                value={sweepConfigRole}
                                onChange={(e) => {
                                  setSweepConfigRole(e.target.value);
                                }}
                              />
                            </CCol>
                            <CCol xs={1}>
                              <CInput
                                type="text"
                                value={sweepConfigRoleCount}
                                onChange={(e) => {
                                  setSweepConfigRoleCount(e.target.value);
                                }}
                              />
                            </CCol>
                            <CCol xs={2}>
                              <Selector
                                values={[...regions, "all"]}
                                valueFunc={(a) => a}
                                displayFunc={(a) => a}
                                value={sweepConfigRoleRegion}
                                onChange={(e) => {
                                  setSweepConfigRoleRegion(e.target.value);
                                }}
                              />
                            </CCol>
                            <CCol xs={4}>
                              <Selector
                                values={instanceTypes}
                                valueFunc={(a) => a.instanceType}
                                displayFunc={(a) =>
                                  `${a.instanceType} (${a.vCPU} / ${a.ram} / ${a.bandwidth})`
                                }
                                value={sweepConfigRoleInstanceType}
                                onChange={(e) => {
                                  setSweepConfigRoleInstanceType(e.target.value);
                                }}
                              />
                            </CCol>
                            <CCol xs={3}>
                              <CButton
                                shape="square"
                                color="primary"
                                onClick={(e) => {
                                  let roleLaunchTemplates = launchTemplates.filter(lt => {
                                    return (
                                      (sweepConfigRoleRegion === 'all' || lt.region === sweepConfigRoleRegion) &&
                                      (lt.instanceType === sweepConfigRoleInstanceType)
                                    )
                                  });



                                  dispatch(
                                    applySweepRunRoleConfig({
                                      role: sweepConfigRole,
                                      agentChoices: roleLaunchTemplates,
                                      count: sweepConfigRoleCount,
                                    })
                                  );
                                }}
                              >
                                Apply
                              </CButton>
                            </CCol>
                          </CRow>
                              <CRow>
                              <CDataTable
                                items={scheduledRunSweepRoles}
                                scopedSlots={{
                                  Delete: (r) => {
                                    return (
                                      <td>
                                        <CButton
                                          size="sm"
                                          className="btn-pill"
                                          style={{ marginRight: "3px" }}
                                          color="danger"
                                          key={r.role}
                                          onClick={(e) => {
                                            dispatch(
                                              deleteScheduledRunSweepRole(
                                                r.Delete.role
                                              )
                                            );
                                          }}
                                        >
                                          Del
                                        </CButton>
                                      </td>
                                    );
                                  },
                                  Agent: (r) => {
                                    return (
                                      <td>
                                        <Selector
                                          values={launchTemplates.map((a) => {
                                            return {
                                              value: `AWS-${a.id}`,
                                              desc: `Boot new AWS agent (${a.description})`,
                                            };
                                          })}
                                          valueFunc={(a) => a.value}
                                          displayFunc={(a) => a.desc}
                                          value={r.Agent.agentChoiceId}
                                          onChange={(e) => {
                                            dispatch(
                                              setScheduledRunSweepRoleAgent(
                                                r.Agent.role,
                                                r.Agent.roleIdx,
                                                e.target.value
                                              )
                                            );
                                          }}
                                        />
                                      </td>
                                    );
                                  },
                                }}
                              />{" "}
                              </CRow>
                            </CCardBody>
                          </CCard>
                        </CCol>
                      )}
                    </CRow>
                  </CCardBody>
                </CCard>
              </CCol>
            </CRow>
          </CForm>
        </CCol>
      </CRow>
      <CRow>
        <CCol xs={{ size: 4, offset: 4 }}>
          <CButton
            block
            shape="square"
            color="primary"
            onClick={(e) => {
              setTestRunEstimate({});
              dispatch(estimateTestRun(setTestRunEstimate));
              toggleConfirm();
            }}
          >
            Schedule
          </CButton>
        </CCol>
      </CRow>

      <CModal show={confirmModal}>
        <CModalHeader><b>Confirm start test</b></CModalHeader>
        <CModalBody>
          {testRunEstimate.instanceHours && <>
            <p>You are about to schedule {testRunEstimate.testruns === 1 ? 'a test run' : `${testRunEstimate.testruns} test runs`}, which will spawn up EC2 instances. Based on your configured roles, sweep parameters and repeating configuration, we've estimated these instance hours will be used:</p>
            <table class="table">
              <thead>
              <tr><th>Instance type</th><th>Total hours</th></tr>
              </thead>
              <tbody>
              {estimatedInstanceHours.map((ih) => <tr><td>{ih.type}</td><td>{ih.hours}</td></tr>)}
              </tbody>
            </table>
            <p>We estimate this will charge your AWS account with <b>at least {numeral(estimatedCharges).format("$ #,##0.00")}</b></p>
          </>}
        </CModalBody>
        <CModalFooter>
          <CButton
            block
            shape="square"
            color="secondary"
            onClick={toggleConfirm}
          >
            Cancel
          </CButton>
          {testRunEstimate.instanceHours && <CButton
            block
            shape="square"
            color="primary"
            onClick={(e) => {
              dispatch(validateAndScheduleTestRun(history));
              toggleConfirm();
            }}
          >
            Yes, schedule
          </CButton>}
        </CModalFooter>
      </CModal>
    </>
  );
};

export default ScheduleTestRun;

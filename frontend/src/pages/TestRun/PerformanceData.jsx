import React, { useState, useEffect } from "react";
import { useDispatch, useSelector } from 'react-redux';
import client from '../../state/apiclient';
import {
    CCol,
    CRow,
    CCard,
    CCardHeader,
    CCardBody,
  } from "@coreui/react";
import Selector from '../../components/Selector';

const PerformanceData = (props) => {
    const selectedArchitecture = useSelector(state => state.architectures.architectures.find((a) => a.id === (props.testRun?.architectureID || 'default')));
    const perfStatsVersion = 4;
    const perfGraphs = useSelector(state => state.testruns.perfGraphs);
    const [perfAgent, setPerfAgent] = useState('');
    const [perfAgent2, setPerfAgent2] = useState('');
    const [perfCommand, setPerfCommand] = useState('');
    const [perfCommand2, setPerfCommand2] = useState('');
    const [perfGraph, setPerfGraph] = useState('');
    const [perfGraph2, setPerfGraph2] = useState('');
  
    return <CCard>
          <CCardHeader>
            <b>Test Run Performance Data</b>
          </CCardHeader>
          <CCardBody>
            <CRow>
              {[0,1].map((i) =>
              <CCol xs={6}>
                <CRow>
                  <CCol xs={4}>Role:</CCol>
                  <CCol xs={8}>
                      <Selector
                                values={props.testRun.roles}
                                valueFunc={a => {
                                  return a.agentID
                                }}
                                displayFunc={(a) => {
                                  var role = selectedArchitecture.roles.find(
                                    (ar) => ar.role === a.role
                                  );
                                  return `${role.title} ${a.roleIdx+1}`
                                }}
                                value={i === 0 ? perfAgent : perfAgent2}
                                onChange={(e) => {
                                  let val = parseInt(e.target.value)
                                  if(i === 0) {
                                    setPerfGraph('')
                                    setPerfCommand('')
                                    setPerfAgent(Number.isNaN(val) ? -1 : val)
                                  } else {
                                    setPerfGraph2('')
                                    setPerfCommand2('')
                                    setPerfAgent2(Number.isNaN(val) ? -1 : val)
                                  }
                                }}
                              />
                  </CCol>
                </CRow>
                <CRow>
                  <CCol xs={4}>Command:</CCol>
                  <CCol xs={8}>
                      <Selector
                                values={props.testRun.executedCommands.filter((ec) => ec.agentID === (i === 0 ? perfAgent : perfAgent2))}
                                valueFunc={ec => ec.commandID}
                                displayFunc={(ec) => ec.description}
                                value={i === 0 ? perfCommand : perfCommand2}
                                onChange={(e) => {
                                  if(i === 0) {
                                    setPerfGraph('')
                                    setPerfCommand(e.target.value)
                                  } else {
                                    setPerfGraph2('')
                                    setPerfCommand2(e.target.value)
                                  }
                                }}
                              />
                  </CCol>
                </CRow>
                <CRow>
                  <CCol xs={4}>Graph:</CCol>
                  <CCol xs={8}>
                      <Selector
                                values={perfGraphs}
                                valueFunc={p => p.id}
                                displayFunc={p => p.name}
                                value={i === 0 ? perfGraph : perfGraph2}
                                onChange={(e) => {
                                  if(i === 0) {
                                    setPerfGraph(e.target.value)
                                  } else {
                                    setPerfGraph2(e.target.value)
                                  }
                                }}
                              />
                  </CCol>
                </CRow>
                <CRow>
                  {i === 0 && perfAgent && perfCommand && perfGraph && <img onClick={(e) => { window.open(`${client.apiUrl}testruns/${props.testRun.id}/plot/perf_${perfGraph}_${perfStatsVersion}_${perfCommand}`) }} style={{cursor:'pointer', width:'100%'}} src={`${client.apiUrl}testruns/${props.testRun.id}/plot/perf_${perfGraph}_${perfStatsVersion}_${perfCommand}`} />}
                  {i === 1 && perfAgent2 && perfCommand2 && perfGraph2 && <img onClick={(e) => { window.open(`${client.apiUrl}testruns/${props.testRun.id}/plot/perf_${perfGraph2}_${perfStatsVersion}_${perfCommand2}`) }} style={{cursor:'pointer', width:'100%'}} src={`${client.apiUrl}testruns/${props.testRun.id}/plot/perf_${perfGraph2}_${perfStatsVersion}_${perfCommand2}`} />}
                </CRow>
              </CCol>)}
            </CRow>
          </CCardBody>
        </CCard>
}

export default PerformanceData;
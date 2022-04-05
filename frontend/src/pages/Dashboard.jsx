import React, { lazy } from "react";
import { CCol, CRow } from "@coreui/react";
import SimpleCard from '../components/SimpleCard';
import TestResult from "../components/TestResult";
import Moment from "react-moment";
import {selectQueuedTestRunCount, selectTestRunLast24hCount, selectNewestTestRun} from '../state/slices/testruns';
import { useSelector } from 'react-redux';

const TopRowCard = props => <CCol xs={3}><SimpleCard {...props}><h1>{props.children}</h1></SimpleCard></CCol>

const Dashboard = (props) => {
  const last24hCount = useSelector(selectTestRunLast24hCount);
  const queuedCount = useSelector(selectQueuedTestRunCount);
  const agentCount = useSelector(state => state.agents.agentCount);
  const onlineUserCount = useSelector(state => state.system.onlineUsers);
  const newestTestRun = useSelector(selectNewestTestRun);
  const initialStateLoaded = useSelector(state => state.system.initialStateLoaded)
  if (!initialStateLoaded) {
    return (
      <CRow>
        <CCol>
          <h1>Waiting for initial system state</h1>
        </CCol>
      </CRow>
    );
  }
  return (
    <>
      <CRow>
        <TopRowCard center="true" title="Executed test runs past 24h">{last24hCount}</TopRowCard>
        <TopRowCard center="true" title="Online agents">{agentCount}</TopRowCard>
        <TopRowCard center="true" title="Queued Test Runs">{queuedCount}</TopRowCard>
        <TopRowCard center="true" title="Connected users">{onlineUserCount}</TopRowCard>
      </CRow>
      <CRow>
        <CCol>
          <SimpleCard center="true" title={<>Most recent test result (
                <Moment format="LL">
                  {newestTestRun.completed}
                </Moment>{" "}
                <Moment format="LT">
                  {newestTestRun.completed}
                </Moment>
                )
              </>}><TestResult testRun={newestTestRun} /></SimpleCard>
        </CCol>
      </CRow>
    </>
  );
};

export default Dashboard;

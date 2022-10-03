import React, { useEffect, useState } from "react";
import {
    CCard,
    CCardBody,
    CCardHeader,
    CCol,
    CRow,
    CButton,
    CInput,
    CTextarea,
    CContainer,
    CCardFooter
} from "@coreui/react";
import { mapListFields, selectSweepRuns, loadSweepRuns } from "../state/slices/testruns"
import { useDispatch, useSelector } from "react-redux";
import { useHistory, useParams } from "react-router-dom";
import client from "../state/apiclient";
import internal from "stream";
import { number } from "prop-types";

const PeakFindingRun = (props) => <CCol xs={12}>
    <CCard>
        <CCardHeader><h2>{props.title}</h2></CCardHeader>
        <CCardBody>
            <p>{props.explainer}</p>
            <p>Ran from <b>{Math.floor(props.run?.loadGenTPSStepStart * props.run?.loadGenTPSTarget)}</b> - <b>{Math.floor(props.run?.loadGenTPSTarget)}</b> tx/s, increasing with <b>{Math.floor(props.run?.loadGenTPSTarget * props.run?.loadGenTPSStepPercent)}</b> tx/s every <b>{Math.floor(props.run?.loadGenTPSStepTime)}</b> seconds</p>
            <p>Registered observed peak as <b>{props.run?.observedPeak}</b> tx/s</p>
        </CCardBody>
        <CCardBody>
            <CRow>
                <CCol xs={4}>
                    <CCard>
                        <CCardHeader>
                            <b>
                                <u>Throughput over time</u>
                            </b>
                        </CCardHeader>
                        <CCardBody>
                            <img
                                style={{ width: "100%" }}
                                src={`${client.apiUrl}testruns/${props.run?.id}/plot/system_throughput_line?v=1`}
                            />
                        </CCardBody>
                    </CCard>
                </CCol>
                <CCol xs={4}>
                    <CCard>
                        <CCardHeader>
                            <b>
                                <u>Latency over time</u>
                            </b>
                        </CCardHeader>
                        <CCardBody>
                            <img
                                style={{ width: "100%" }}
                                src={`${client.apiUrl}testruns/${props.run?.id}/plot/system_latency_line?v=1`}
                            />
                        </CCardBody>
                    </CCard>
                </CCol>
                <CCol xs={4}>
                    <CCard>
                        <CCardHeader>
                            <b>
                                <u>Latency elbow plot</u>
                            </b>
                        </CCardHeader>
                        <CCardBody>
                            <img
                                style={{ width: "100%" }}
                                src={`${client.apiUrl}testruns/${props.run?.id}/plot/system_elbow_plot?v=1`}
                            />
                        </CCardBody>
                    </CCard>
                </CCol>
            </CRow>
        </CCardBody>
    </CCard>
</CCol>;

const ConfirmationRuns = (props) => <CCol xs={12}>
    <CCard>
        <CCardHeader><h2>{props.title}</h2></CCardHeader>
        <CCardBody>
            <p>{props.explainer}</p>
            <p>Ran <b>{props.runs.length}</b> runs for <b>{props.runs[0].sampleCount}</b> seconds at <b>{props.runs[0].loadGenTPSTarget}</b> tx/s</p>
        </CCardBody>
        <CCardBody>
            <CRow>
                {props.runs.map((r, i) =>
                    <CCol xs={4}>
                        <CCard>
                            <CCardHeader>
                                <b>
                                    <u>Throughput over time run {i + 1} ({r.id})</u>
                                </b>
                            </CCardHeader>
                            <CCardBody>
                                <img
                                    style={{ width: "100%" }}
                                    src={`${client.apiUrl}testruns/${r.id}/plot/system_throughput_line?v=1`}
                                />
                            </CCardBody>
                        </CCard>
                    </CCol>)}
            </CRow>
            <CRow>
                {props.runs.map((r, i) => <CCol xs={4}>
                    <CCard>
                        <CCardHeader>
                            <b>
                                <u>Latency over time run {i + 1} ({r.id})</u>
                            </b>
                        </CCardHeader>
                        <CCardBody>
                            <img
                                style={{ width: "100%" }}
                                src={`${client.apiUrl}testruns/${r.id}/plot/system_latency_line?v=1`}
                            />
                        </CCardBody>
                    </CCard>
                </CCol>)}
            </CRow>
        </CCardBody>
    </CCard>
</CCol>;



const PeakFindingResult = (props) => {
    const architectures = useSelector(
        (state) => state.architectures?.architectures
    );
    const users = useSelector(
        (state) => state.users?.users
    );
    const params = useParams();
    const sweep = useSelector(state => state.testruns.sweeps.find((s) => s.id === params.sweepID));
    const dispatch = useDispatch();
    const sweepRuns = useSelector(state => selectSweepRuns(state, params.sweepID));
    const initialStateLoaded = useSelector(state => state.system.initialStateLoaded)

    const testRunSummary = mapListFields(architectures, users);

    useEffect(() => {
        if (sweep && sweep.runsAvailable !== true && sweep.runsLoading !== true) {
            dispatch(loadSweepRuns(sweep.id));
        }
    }, [sweep]);

    if (!initialStateLoaded) {
        return (
            <CRow>
                <CCol>
                    <h1>Waiting for initial system state</h1>
                </CCol>
            </CRow>
        );
    }


    if (!sweep || sweep.id !== params.sweepID) {
        return (
            <CRow>
                <CCol>
                    <h1>Not found</h1>
                </CCol>
            </CRow>
        );
    }

    if (sweep.runsAvailable !== true) {
        return (
            <CRow>
                <CCol>
                    <h1>Loading...</h1>
                </CCol>
            </CRow>
        );
    }

    for (let run of sweepRuns) {
        if (run.detailsAvailable !== true) {
            return (
                <CRow>
                    <CCol>
                        <h1>Loading test run {run.id}...</h1>
                    </CCol>
                </CRow>
            );
        }
    }

    const firstRun = sweepRuns.find(r => r.status === "Completed" && r.loadGenTPSStepStart === 0)
    const secondRun = sweepRuns.find(r => r.status === "Completed" && r.loadGenTPSStepStart !== 0 && r.loadGenTPSStepStart !== 1)

    const confirmationRuns = sweepRuns.filter(r => r.status === "Completed" && r.loadGenTPSStepStart === 1).sort((a, b) => a.loadGenTPSTarget - b.loadGenTPSTarget);

    const summary = testRunSummary(Object.assign({}, firstRun, { loadGenTPSTarget: 0 }));

    const peakConfirm = confirmationRuns.filter(r => r.loadGenTPSTarget === confirmationRuns[0].loadGenTPSTarget);

    const overloadRuns = confirmationRuns.filter(r => r.loadGenTPSTarget !== peakConfirm[0].loadGenTPSTarget);

    let overloadSets = [];

    for (let run of overloadRuns) {
        let idx = overloadSets.findIndex(s => s.tpsTarget === run.loadGenTPSTarget);
        if(idx === -1) {
            overloadSets.push({tpsTarget: run.loadGenTPSTarget, runs: [run]})
        } else {
            overloadSets[idx].runs = [...overloadSets[idx].runs, run];
        }
    }

    
    return (
        <>
            <CRow alignHorizontal="center">
                <h1>Peak finding sweep {params.sweepID} - result</h1>
            </CRow>
            <CRow alignHorizontal="center">
                <h2>{summary.roles}</h2>
            </CRow>
            <CRow alignHorizontal="center">
                <h4>{summary.params} / Commit: {firstRun.commitHash.substr(0,7)}</h4>
            </CRow>
            <CRow alignHorizontal="center">
                <h2>Confirmed peak: <b>{peakConfirm[0]?.loadGenTPSTarget} tx/s</b></h2>
            </CRow>
            <CRow alignHorizontal="center">
                <h4>Confirmed overload at: <b>{overloadSets[0]?.tpsTarget} tx/s</b></h4>
            </CRow>
            <CRow>
                <PeakFindingRun title="First estimation run" explainer="In the first estimation run, the system increases the load generator throttle from 0 tx/s to about 150% of the expected peak. The throughput plot should show the throttle ('Loadgen Target') rising and the registered througput ('Loadgens') top out below the generated traffic. The latency time series should show a growing latency at the end, confirming that we surpassed the peak the system can handle. From these plots, we manually observe a peak throughput that the system can still handle, which is the input to the second estimation run." run={firstRun} />
                <PeakFindingRun title="Second estimation run" explainer="In the second estimation run, the system increases the load generator throttle from 90% to 110% of the peak observed in the first run. The throughput and latency plots should show the same pattern as before, but by zooming in to the range around the observed peak, we can get a more precise estimation of the peak. From the plots, we manually observe the peak throughput again, and use it as the input to the confirmation runs" run={secondRun} />
                <ConfirmationRuns title="Peak Confirmation" explainer="In order to confirm the peak, we run the system at 95% of the observed maximum in the second estimation run. We take this margin because the performance seems to be varying due to using virtualized underlying hardware. System performance and bandwidth are not 100% guaranteed and can fluctuate. By selecting a peak slightly below the observed maximum in a single run, the chance of confirming this as a peak the system can consistently handle within the latency constraints is much higher. If variance in performance affects one of the confirmation runs, the confirmation fails. The three runs should show constant throughput and constant latency with possible peaks staying within the latency constraints." runs={peakConfirm} />
                {overloadSets.map(s => <ConfirmationRuns title={`Overload Confirmation @ ${s.tpsTarget} tx/s`} explainer="In order to confirm at what level the system is overloaded, we run the system at 105% of the observed maximum in the second estimation run, due to variance in performance. At least one of the three runs should show a growing latency and/or peaks exceeding the latency constraints." runs={s.runs} />)}
            </CRow>
        </>
    );
};

export default PeakFindingResult;

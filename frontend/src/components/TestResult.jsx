import React, { useState } from "react";
import "./CommandOutput.css";
import * as numeral from "numeral";
import {
  CButton,
  CCol,
  CRow,
  CCard,
  CCardHeader,
  CCardBody,
  CButtonGroup,
  CFormGroup,
  CLabel,
  CInput,
  CInputCheckbox,
} from "@coreui/react";
import { reloadTestResults, redownloadOutputs, confirmPeak } from "../state/slices/testruns";
import { useDispatch } from "react-redux";
import client from "../state/apiclient";

const TestResult = (props) => {
  const [trimZeroes, setTrimZeroes] = useState(props.testRun.trimZeroesAtStart);
  const [trimZeroesEnd, setTrimZeroesEnd] = useState(props.testRun.trimZeroesAEnd);
  const [trimSamples, setTrimSamples] = useState(props.testRun.trimSamplesAtStart);
  const [observedPeak, setObservedPeak] = useState(props.testRun?.observedPeak);
  const [peakRerun, setPeakRerun] = useState(false);
  const lastRecalc = props.testRun.resultUpdated?.valueOf() || 0;

  const dispatch = useDispatch();
  return (
    <>
      {props.testRun?.result && (
        <>
          
          <CRow>
            <CCol xs={6}>
              <CCard>
                <CCardHeader>Result trimming</CCardHeader>
                <CCardBody>
                  <CFormGroup row>
                    <CCol sm={6}>
                      <CLabel htmlFor="trimSamples">Trim Samples:</CLabel>
                    </CCol>
                    <CCol sm={6}>
                      <CInput
                        type="text"
                        id="trimSamples"
                        value={trimSamples}
                        onChange={(e) => {
                          let val = parseInt(e.target.value);
                          if (Number.isNaN(val)) val = 0;
                          setTrimSamples(val);
                        }}
                      ></CInput>
                    </CCol>
                  </CFormGroup>
                  <CFormGroup row>
                    <CCol sm={6}>
                      <CLabel htmlFor="trimZeroes">Trim Zeroes at start:</CLabel>
                    </CCol>
                    <CCol sm={6}>
                      <CInputCheckbox
                        id="trimZeroes"
                        checked={trimZeroes}
                        onChange={(e) => {
                          setTrimZeroes(e.target.checked);
                        }}
                      ></CInputCheckbox>
                    </CCol>
                  </CFormGroup>
                  <CFormGroup row>
                    <CCol sm={6}>
                      <CLabel htmlFor="trimZeroes">Trim Zeroes at end:</CLabel>
                    </CCol>
                    <CCol sm={6}>
                      <CInputCheckbox
                        id="trimZeroesEnd"
                        checked={trimZeroesEnd}
                        onChange={(e) => {
                          setTrimZeroesEnd(e.target.checked);
                        }}
                      ></CInputCheckbox>
                    </CCol>
                  </CFormGroup>
                </CCardBody>
              </CCard>
            </CCol>
            <CCol xs={{ size: 4, offset: 1 }} style={{ paddingBottom: "10px" }}>
              <CButton
                onClick={(e) => {
                  dispatch(reloadTestResults(props.testRun?.id, trimZeroes, trimSamples, trimZeroesEnd));
                }}
                size="sm"
                className="btn-pill"
                block
                color="primary"
              >
                Recalculate Test Results
              </CButton><br />
              <CButton
                onClick={(e) => {
                  dispatch(redownloadOutputs(props.testRun?.id));
                }}
                size="sm"
                className="btn-pill"
                block
                color="primary"
              >
                Re-download Test Outputs from S3
              </CButton>
            </CCol>
          </CRow>
          <CRow>
            <CCol xs={6}>
              <CCard>
                <CCardHeader>
                  <b>
                    <u>Throughput results</u>
                  </b>
                </CCardHeader>
                <CCardBody>
                  <CRow>
                    <CCol xs={4}>Average:</CCol>
                    <CCol xs={8}>
                      <b>
                        {numeral(props.testRun.result.throughputAvg).format(
                          "#,##0.00"
                        )}{" "}
                        tx/s
                      </b>
                    </CCol>
                  </CRow>
                  {props.testRun.result.throughputAvg2 && props.testRun.result.throughputAvg2 > 0 && <CRow>
                    <CCol xs={4}>Average (2):</CCol>
                    <CCol xs={8}>
                      <b>
                        {numeral(props.testRun.result.throughputAvg2).format(
                          "#,##0.00"
                        )}{" "}
                        tx/s
                      </b>
                    </CCol>
                  </CRow>}
                  <CRow>
                    <CCol xs={4}>Std Dev:</CCol>
                    <CCol xs={8}>
                      <b>
                        {numeral(props.testRun.result.throughputStd).format(
                          "#,##0.00"
                        )}{" "}
                        tx/s
                      </b>
                    </CCol>
                  </CRow>
                  <CRow>
                    <CCol xs={4}>Min:</CCol>
                    <CCol xs={8}>
                      <b>
                        {numeral(props.testRun.result.throughputMin).format(
                          "#,##0.00"
                        )}{" "}
                        tx/s
                      </b>
                    </CCol>
                  </CRow>
                  <CRow>
                    <CCol xs={4}>Max:</CCol>
                    <CCol xs={8}>
                      <b>
                        {numeral(props.testRun.result.throughputMax).format(
                          "#,##0.00"
                        )}{" "}
                        tx/s
                      </b>
                    </CCol>
                  </CRow>
                  <CRow>
                    <CCol xs={4}>
                      <u>Percentiles:</u>
                    </CCol>
                    <CCol xs={8}>&nbsp;</CCol>
                  </CRow>
                  {props.testRun.result.throughputPercentiles.map((pct) => (
                    <CRow key={pct.bucket}>
                      <CCol xs={4}>&nbsp;&nbsp;{pct.bucket}:</CCol>
                      <CCol xs={8}>
                        <b>{numeral(pct.value).format("#,##0.00")} tx/s</b>
                      </CCol>
                    </CRow>
                  ))}
                </CCardBody>
              </CCard>
            </CCol>
            <CCol xs={6}>
              <CCard>
                <CCardHeader>
                  <b>
                    <u>Latency results</u>
                  </b>
                </CCardHeader>
                <CCardBody>
                  <CRow>
                    <CCol xs={4}>Average:</CCol>
                    <CCol xs={8}>
                      <b>
                        {numeral(
                          props.testRun.result.latencyAvg * Math.pow(10, 3)
                        ).format("#,##0.00")}{" "}
                        ms
                      </b>
                    </CCol>
                  </CRow>
                  <CRow>
                    <CCol xs={4}>Std Dev:</CCol>
                    <CCol xs={8}>
                      <b>
                        {numeral(
                          props.testRun.result.latencyStd * Math.pow(10, 3)
                        ).format("#,##0.00")}{" "}
                        ms
                      </b>
                    </CCol>
                  </CRow>
                  <CRow>
                    <CCol xs={4}>Min:</CCol>
                    <CCol xs={8}>
                      <b>
                        {numeral(
                          props.testRun.result.latencyMin * Math.pow(10, 3)
                        ).format("#,##0.00")}{" "}
                        ms
                      </b>
                    </CCol>
                  </CRow>
                  <CRow>
                    <CCol xs={4}>Max:</CCol>
                    <CCol xs={8}>
                      <b>
                        {numeral(
                          props.testRun.result.latencyMax * Math.pow(10, 3)
                        ).format("#,##0.00")}{" "}
                        ms
                      </b>
                    </CCol>
                  </CRow>
                  <CRow>
                    <CCol xs={4}>
                      <u>Percentiles:</u>
                    </CCol>
                    <CCol xs={8}>&nbsp;</CCol>
                  </CRow>
                  {props.testRun.result.latencyPercentiles.map((pct) => (
                    <CRow key={pct.bucket}>
                      <CCol xs={4}>&nbsp;&nbsp;{pct.bucket}:</CCol>
                      <CCol xs={8}>
                        <b>
                          {numeral(pct.value * Math.pow(10, 3)).format(
                            "#,##0.00"
                          )}{" "}
                          ms
                        </b>
                      </CCol>
                    </CRow>
                  ))}
                </CCardBody>
              </CCard>
            </CCol>
          </CRow>
          <CRow>
            <CCol xs={4}>
              <CCard>
                <CCardHeader>
                  <b>
                    <u>Throughput distribution</u>
                  </b>
                </CCardHeader>
                <CCardBody>
                  <img
                    style={{ width: "100%" }}
                    src={`${client.apiUrl}testruns/${props.testRun.id}/plot/system_throughput_hist?v=${lastRecalc}`}
                  />
                </CCardBody>
              </CCard>
            </CCol>
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
                    src={`${client.apiUrl}testruns/${props.testRun.id}/plot/system_throughput_line?v=${lastRecalc}`}
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
                    src={`${client.apiUrl}testruns/${props.testRun.id}/plot/system_latency_line?v=${lastRecalc}`}
                  />
                </CCardBody>
              </CCard>
            </CCol>
            <CCol xs={4}>
              <CCard>
                <CCardHeader>
                  <b>
                    <u>Latency distribution</u>
                  </b>
                </CCardHeader>
                <CCardBody>
                  <img
                    style={{ width: "100%" }}
                    src={`${client.apiUrl}testruns/${props.testRun.id}/plot/system_latency_hist?v=${lastRecalc}`}
                  />
                </CCardBody>
              </CCard>
            </CCol>
            <CCol xs={4}>
              <CCard>
                <CCardHeader>
                  <b>
                    <u>Latency distribution</u>
                  </b>
                </CCardHeader>
                <CCardBody>
                  <img
                    style={{ width: "100%" }}
                    src={`${client.apiUrl}testruns/${props.testRun.id}/plot/system_elbow_plot?v=${lastRecalc}`}
                  />
                </CCardBody>
              </CCard>
            </CCol>
            {props.testRun?.sweep === 'peak' && props.testRun?.loadGenTPSStepStart !== 1 && <CCol xs={4}>
              <CCard>
                <CCardHeader><b><u>Observed Peak Bandwidth</u></b></CCardHeader>
                <CCardBody>
                  <p>Observe the elbow plot and determine where the latency starts rising. Enter this value here.</p>
                  {props.testRun?.loadGenTPSStepStart === 0 && <p>A second run will be started that ranges from 90-110% of the entered figure, rounded to the nearest 100 TPS.</p>}
                  {props.testRun?.loadGenTPSStepStart !== 0 && <p>Peak confirmation runs will be scheduled that are throttled at 5% below and 5% above the entered figure, rounded to the nearest 100 TPS.</p>}
                  <CFormGroup row>
                    <CCol sm={6}>
                      <CLabel htmlFor="observedPeak">Observed Peak TPS:</CLabel>
                    </CCol>
                    <CCol sm={6}>
                      <CInput
                        type="text"
                        id="observedPeak"
                        value={observedPeak}
                        onChange={(e) => {
                          let val = parseInt(e.target.value);
                          if (Number.isNaN(val)) val = 0;
                          setObservedPeak(val);
                        }}
                      ></CInput>
                    </CCol>
                  </CFormGroup>
                  <CFormGroup row>
                    <CCol sm={6}>
                      <CLabel htmlFor="peakUB">Force Rerun Confirmation Runs:<br/><small>If the peak was entered before but the confirmation runs ended up invalid, check this box to force the confirmation runs to run again.</small></CLabel>
                    </CCol>
                    <CCol sm={6}>
                    <CInputCheckbox
                        id="forceRerun"
                        checked={peakRerun}
                        onChange={(e) => {
                          setPeakRerun(e.target.checked);
                        }}
                      ></CInputCheckbox>           

                    </CCol>
                  </CFormGroup>
                  <CFormGroup row>
                    <CCol sm={12}>
                      <CButton
                        onClick={(e) => {
                          dispatch(confirmPeak(props.testRun?.id, observedPeak, peakRerun));
                        }}
                        size="sm"
                        className="btn-pill"
                        block
                        color="primary"
                      >
                        Confirm Peak
                      </CButton>
                    </CCol>
                  </CFormGroup>
                </CCardBody>
              </CCard>
            </CCol>}
          </CRow>
        </>
      )}
      {!props.testRun?.result && (
        <CRow>
          <CCol xs={6}>Results not (yet) available</CCol>
          <CCol xs={6}><CButton
            onClick={(e) => {
              dispatch(reloadTestResults(props.testRun?.id, trimZeroes, trimSamples, trimZeroesEnd));
            }}
            size="sm"
            className="btn-pill"
            block
            color="primary"
          >
            Recalculate Test Results
          </CButton></CCol>
        </CRow>
      )}
    </>
  );
};

export default TestResult;

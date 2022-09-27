import Moment from "react-moment";
import {
  CDropdown,
  CDropdownToggle,
  CDropdownMenu,
  CDropdownItem,
  CButton,
  CCol,
  CRow,
  CDataTable,
} from "@coreui/react";
import { useHistory } from "react-router-dom";
import { useDispatch, useSelector } from "react-redux";
import {
  continueOneAtATimeSweep,
  loadTestRunDetails,
  terminateTestRunSweep,
  selectActiveTestRunsList,
  selectCompletedTestRunsList,
  selectQueuedTestRunsList,
  selectFailedTestRunsList,
  rescheduleTestRun,
  terminateTestRun,
} from "../state/slices/testruns";
import "./TestRuns.css";
import { _ } from "core-js";

const TestRuns = (props) => {
  const history = useHistory();
  const queuedTestRuns = useSelector(selectQueuedTestRunsList);
  const activeTestRuns = useSelector(selectActiveTestRunsList);
  const completedTestRuns = useSelector(selectCompletedTestRunsList);
  const failedTestRuns = useSelector(selectFailedTestRunsList);
  const dispatch = useDispatch();
  var testRuns = [];
  var fields = [];
  const initialStateLoaded = useSelector(
    (state) => state.system.initialStateLoaded
  );
  if (!initialStateLoaded) {
    return (
      <CRow>
        <CCol>
          <h1>Waiting for initial system state</h1>
        </CCol>
      </CRow>
    );
  }

  var scopedSlots = {
    params: (r) => (
      <td>
        {r.params.split(" / ").map((p) => (
          <>
            {p}
            <br />
          </>
        ))}
      </td>
    ),
    actions: (r) => (
      <td>
        <CDropdown variant="btn-group">
          <CButton
            color="primary"
            onClick={(e) => {
              if (r.detailsAvailable !== true) {
                dispatch(loadTestRunDetails(r.id));
              }
              history.push(`/testrun/${r.id}`);
            }}
          >
            Details
          </CButton>
          <CDropdownToggle color="primary" split />
          <CDropdownMenu>
            <CDropdownItem
              onClick={(e) => {
                dispatch(rescheduleTestRun(r.id, false));
                history.push(`/testruns/schedule`);
              }}
            >
              Schedule again
            </CDropdownItem>

            {r.sweepID && r.sweepID !== "" && (
              <CDropdownItem
                onClick={(e) => {
                  dispatch(rescheduleTestRun(r.id, true));
                  history.push(`/testruns/schedule`);
                }}
              >
                Schedule sweep again
              </CDropdownItem>
            )}
            {r.status === "Completed" &&
              r.sweepID &&
              r.sweepOneAtATime === true &&
              r.sweepID !== "" && (
                <CDropdownItem
                  onClick={(e) => {
                    dispatch(continueOneAtATimeSweep(r.sweepID));
                  }}
                >
                  Resume sweep
                </CDropdownItem>
              )}
            {(r.status === "Failed" || r.status === "Aborted") &&
              r.sweepID &&
              r.sweepOneAtATime === true &&
              r.sweepID !== "" && (
                <CDropdownItem
                  onClick={(e) => {
                    dispatch(continueOneAtATimeSweep(r.sweepID));
                  }}
                >
                  Retry and resume sweep
                </CDropdownItem>
              )}
            {r.status === "Queued" && (
              <CDropdownItem
                onClick={(e) => {
                  dispatch(terminateTestRun(r.id));
                }}
              >
                Cancel
              </CDropdownItem>
            )}
            {r.status === "Running" && (
              <CDropdownItem
                onClick={(e) => {
                  dispatch(terminateTestRun(r.id));
                }}
              >
                Terminate
              </CDropdownItem>
            )}
            {r.status === "Queued" && r.sweepID && r.sweepID !== "" && (
              <CDropdownItem
                onClick={(e) => {
                  dispatch(terminateTestRunSweep(r.sweepID));
                }}
              >
                Cancel Sweep
              </CDropdownItem>
            )}
            {r.status === "Running" && r.sweepID && r.sweepID !== "" && (
              <CDropdownItem
                onClick={(e) => {
                  dispatch(terminateTestRunSweep(r.sweepID));
                }}
              >
                Terminate Sweep
              </CDropdownItem>
            )}
          </CDropdownMenu>
        </CDropdown>
      </td>
    ),
  };

  var fields = [
    { key: "id", label: "ID" },
    { key: "sweepID", label: "Sweep" },
    { key: "architectureName", label: "Architecture" },
    { key: "roles", label: "Roles" },
    { key: "sortDate", label: "Status" },
    { key: "params", label: "Params" },
  ];

  if (props.match.params.state === "running") {
    testRuns = activeTestRuns;
    fields.push({ key: "deets", label: "Details" });
    scopedSlots["sortDate"] = (r) => (
      <td>
        {r.status}{" "}
        {!r.started.startsWith("0001") && (
          <>
            <Moment format="L">{r.started}</Moment>{" "}
            <Moment format="LT">{r.started}</Moment>
          </>
        )}
      </td>
    );
    scopedSlots["deets"] = (r) => (
      <td style={{ width: "400px" }}>
        <div
          style={{
            width: "400px",
            overflow: "hidden",
            whiteSpace: "nowrap",
            textOverflow: "ellipsis",
          }}
        >
          {r.details}
        </div>
      </td>
    );
  } else if (props.match.params.state === "queued") {
    testRuns = queuedTestRuns;
    fields.push({ key: "deets", label: "Details" });
    scopedSlots["sortDate"] = (r) => (
      <td>
        {r.status}{" "}
        {r.status === "Queued" && !r.notBefore.startsWith("0001") && (
          <>
            until{" "}
            <>
              <Moment format="L">{r.notBefore}</Moment>{" "}
              <Moment format="LT">{r.notBefore}</Moment>
            </>
          </>
        )}{" "}
      </td>
    );
    scopedSlots["deets"] = (r) => (
      <td style={{ width: "400px" }}>
        <div
          style={{
            width: "400px",
            overflow: "hidden",
            whiteSpace: "nowrap",
            textOverflow: "ellipsis",
          }}
        >
          {r.details}
        </div>
      </td>
    );
  } else if (props.match.params.state === "completed") {
    fields.push({ key: "avgThroughput", label: "Throughput (avg)" });
    fields.push({ key: "tailLatency", label: "99% latency" });
    scopedSlots["sortDate"] = (r) => (
      <td>
        Completed <Moment format="L">{r.completed}</Moment>{" "}
        <Moment format="LT">{r.completed}</Moment>
      </td>
    );
    scopedSlots["avgThroughput"] = (r) => <td>{`${Math.floor(r.avgThroughput)} tx/s`}</td>;
    scopedSlots["tailLatency"] = (r) => <td>{`${Math.floor(r.tailLatency * 100000) / 100} ms`}</td>;
    testRuns = completedTestRuns;
  } else if (props.match.params.state === "failed") {
    fields.push({ key: "deets", label: "Details" });
    scopedSlots["sortDate"] = (r) => (
      <td>
        {r.status}{" "}
        {!r.completed.startsWith("0001") && (
          <>
            <Moment format="L">{r.completed}</Moment>{" "}
            <Moment format="LT">{r.completed}</Moment>
          </>
        )}
      </td>
    );
    scopedSlots["deets"] = (r) => (
      <td style={{ width: "400px" }}>
        <div
          style={{
            width: "400px",
            overflow: "hidden",
            whiteSpace: "nowrap",
            textOverflow: "ellipsis",
          }}
        >
          {r.details}
        </div>
      </td>
    );
    testRuns = failedTestRuns;
  } else if (props.match.params.state === "pendingPeakObservation") {
    fields.push({ key: "avgThroughput", label: "Throughput (avg)" });
    fields.push({ key: "tailLatency", label: "99% latency" });
    scopedSlots["sortDate"] = (r) => (
      <td>
        Completed <Moment format="L">{r.completed}</Moment>{" "}
        <Moment format="LT">{r.completed}</Moment>
      </td>
    );
    scopedSlots["avgThroughput"] = (r) => <td>{`${Math.floor(r.avgThroughput)} tx/s`}</td>;
    scopedSlots["tailLatency"] = (r) => <td>{`${Math.floor(r.tailLatency * 100000) / 100} ms`}</td>;
    testRuns = completedTestRuns.filter(r => r.sweep === "peak" && r.loadGenTPSStepStart !== 1 && r.observedPeak === 0);
  } else if (props.match.params.state === "sweep") {
    fields.push({ key: "avgThroughput", label: "Throughput (avg)" });
    fields.push({ key: "tailLatency", label: "99% latency" });
    scopedSlots["sortDate"] = (r) => (
      <td>
        Completed <Moment format="L">{r.completed}</Moment>{" "}
        <Moment format="LT">{r.completed}</Moment>
      </td>
    );
    scopedSlots["avgThroughput"] = (r) => <td>{`${Math.floor(r.avgThroughput)} tx/s`}</td>;
    scopedSlots["tailLatency"] = (r) => <td>{`${Math.floor(r.tailLatency * 100000) / 100} ms`}</td>;
    testRuns = completedTestRuns.filter(r => r.sweepID === props.match.params.spec);
  }

  fields.push({ key: "actions", label: "" });

  return (
    <>
      <CRow>
        <CCol>
          <h1>Test runs ({props.match.params.state === 'sweep' ? `Sweep ${props.match.params.spec}` : (props.match.params.state === 'pendingPeakObservation' ? "Pending peak observation" : props.match.params.state)})</h1>
          <CDataTable
            striped
            pagination={{ align: "center" }}
            items={testRuns}
            fields={fields}
            columnFilter={true}
            sorter={true}
            sorterValue={{ column: "sortDate", asc: false }}
            scopedSlots={scopedSlots}
          />
        </CCol>
      </CRow>
    </>
  );
};

export default TestRuns;

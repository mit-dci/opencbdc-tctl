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
  CInputCheckbox,
} from "@coreui/react";
import { useHistory } from "react-router-dom";
import { useDispatch, useSelector } from "react-redux";
import {
  rescheduleMissingSweepRuns,
  selectSweeps,
} from "../state/slices/testruns";
import { _ } from "core-js";
import { useState } from "react";

const Sweeps = (props) => {
  const history = useHistory();
  const dispatch = useDispatch();

  const sweeps = useSelector(selectSweeps);
  const [multiSweep, setMultiSweep] = useState([]);
  const architectures = useSelector(
    (state) => state.architectures?.architectures
  );
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
    check: (r) => (
      <td style={{ textAlign: "left" }}>
        <CInputCheckbox
          style={{ marginLeft: 0 }}
          checked={multiSweep.indexOf(r.id) > -1}
          onChange={(e) => {
            var newMultiSweep = multiSweep.filter((s) => s !== r.id);
            if (e.target.checked) {
              newMultiSweep.push(r.id);
            }
            newMultiSweep.sort();
            setMultiSweep(newMultiSweep);
          }}
        />
      </td>
    ),
    actions: (r) => (
      <td>
        <CDropdown variant="btn-group">
          <CButton
            color="primary"
            onClick={(e) => {
              history.push(`/testruns/sweepMatrix/${r.id}`);
            }}
          >
            Matrix
          </CButton>
          <CDropdownToggle color="primary" split />
          <CDropdownMenu>
            <CDropdownItem
             
              onClick={(e) => {
                history.push(`/sweepPlot/${r.id}`);
              }}
            >
              Plots
            </CDropdownItem>
            <CDropdownItem
              
              onClick={(e) => {
                dispatch(rescheduleMissingSweepRuns(r.id));
              }}
            >
              Reschedule missing
            </CDropdownItem>
          </CDropdownMenu>
        </CDropdown>
      </td>
    ),
    params: (r) => {
      let roleDesc = "";
      if (r.sweepType === "roles") {
        let roles = {};
        let arch = architectures.find(
          (a) => a.id === (r.architectureID || "default")
        ) || {roles:[]};
        for (var role of r.sweepRoles) {
          if (roles[role.role]) {
            roles[role.role] = roles[role.role] + 1;
          } else {
            roles[role.role] = 1;
          }
        }
        roleDesc = Object.keys(roles)
          .map((k) => {
            var role = arch.roles.find((r) => r.role === k);
            return `${roles[k]} ${role ? role.shortTitle : k.substr(0, 4)}`;
          })
          .join(" / ");
      }
      return (
        <td>
          {r.sweepType === "parameter" &&
            `${r.sweepParameter} ${r.sweepParameterStart}...${r.sweepParameterStop} (step: ${r.sweepParameterIncrement})`}
          {r.sweepType === "roles" &&
            `${r.sweepRoleRuns} total runs - add ${roleDesc} each run`}
        </td>
      );
    },
    firstRun: (r) => (
      <td>
        <Moment format="LL">{r.firstRun}</Moment>{" "}
        <Moment format="LT">{r.firstRun}</Moment>
      </td>
    ),
    lastRun: (r) => (
      <td>
        <Moment format="LL">{r.lastRun}</Moment>{" "}
        <Moment format="LT">{r.lastRun}</Moment>
      </td>
    ),
  };

  var fields = [
    { key: "check", label: "Select" },
    { key: "id", label: "ID" },
    { key: "architectureID", label: "Architecture" },
    { key: "firstRun", label: "First Run" },
    { key: "roles", label: "Roles" },
    { key: "sweepType", label: "Sweep Type" },
    { key: "params", label: "Parameters" },
    { key: "runCount", label: "Run Count" },
    { key: "fixedTxRate", label: "Fixed" },
    { key: "invalidTxRate", label: "Invalid" },
    { key: "loadGenInputCount", label: "Inputs" },
    { key: "loadGenOutputCount", label: "Outputs" },
    { key: "preseedCount", label: "Preseed" },
    { key: "shardReplicationFactor", label: "Shard Repl" },
    { key: "actions", label: "" },
  ];

  return (
    <>
      <CRow>
        <CCol>
          <div style={{ float: "left" }}>
            <h1>Sweeps</h1>
          </div>
          <div style={{ float: "right" }}>
            {multiSweep.length > 0 && (
              <CButton
                color="primary"
                onClick={(e) => {
                  history.push(`/sweepPlot/${multiSweep.join("|")}`);
                }}
              >
                Create plot from {multiSweep.length} sweeps
              </CButton>
            )}{" "}
            {multiSweep.length > 0 && (
              <CButton
                color="primary"
                onClick={(e) => {
                  history.push(`/testruns/sweepMatrix/${multiSweep.join("|")}`);
                }}
              >
                Combined matrix of {multiSweep.length} sweeps
              </CButton>
            )}
          </div>
          <CDataTable
            striped
            items={sweeps}
            fields={fields}
            pagination={{ align: "center" }}
            columnFilter={true}
            scopedSlots={scopedSlots}
            sorter={true}
            sorterValue={{ column: "firstRun", asc: false }}
          />
        </CCol>
      </CRow>
    </>
  );
};

export default Sweeps;

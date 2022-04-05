import React, { useState } from "react";
import { useHistory } from "react-router-dom";
import {
  CContainer,
  CCard,
  CCardBody,
  CCardHeader,
  CCol,
  CRow,
  CButton,
  CInput,
  CDataTable,
  CInputFile,
  CForm,
  CLabel,
  CFormGroup,
} from "@coreui/react";
import { setMaxAgents } from "../state/slices/system";
import { useSelector, useDispatch } from "react-redux";
import { toggleMaintenanceMode } from "../state/slices/system";
import { addUser, deleteUser } from "../state/slices/users";

const Config = () => {
  const history = useHistory();
  const [newUser, setNewUser] = useState(null);
  const dispatch = useDispatch();
  const users = useSelector((state) => state.users.users);
  const maintenanceMode = useSelector((state) => state.system?.maintenanceMode);
  const config = useSelector((state) => state.system?.config);
  const [newMaxAgents, setNewMaxAgents] = useState(config?.maxAgents || 500);

  return (
    <CContainer>
      <CRow>
        <CCol xl={6}>
          <CCard>
            <CCardHeader>Agent limit</CCardHeader>
            <CCardBody>
              <CRow>
                <CCol xs={6}>Max active test agents:</CCol>
                <CCol xs={6}>
                  <CInput
                    type="text"
                    value={newMaxAgents}
                    onChange={(e) => {
                      let newVal = parseInt(e.target.value);
                      setNewMaxAgents(Number.isNaN(newVal) ? 0 : newVal);
                    }}
                  ></CInput>
                </CCol>
              </CRow>
              <CRow>
                <CCol xs={12}>
                  <CButton
                    color="primary"
                    onClick={(e) => {
                      setMaxAgents(newMaxAgents);
                    }}
                  >
                    Save
                  </CButton>
                </CCol>
              </CRow>
            </CCardBody>
          </CCard>
        </CCol>

        <CCol xl={6}>
          <CCard>
            <CCardHeader>Maintenance Mode</CCardHeader>
            <CCardBody>
              <CRow>
                <CCol xs={6}>Maintenance mode:</CCol>
                <CCol xs={6}>{maintenanceMode === true ? "On" : "Off"}</CCol>
              </CRow>
              <CRow>
                <CCol xs={12}>
                  <CButton color="primary" onClick={toggleMaintenanceMode}>
                    Toggle
                  </CButton>
                </CCol>
              </CRow>
            </CCardBody>
          </CCard>
        </CCol>
      </CRow>

      <CRow>
        <CCol xl={6}>
          <CCard>
            <CCardHeader>Users</CCardHeader>
            <CCardBody>
              <CDataTable
                items={users}
                sorter={false}
                fields={[
                  { key: "name", _classes: "font-weight-bold", label: "Name" },
                  { key: "email", label: "E-mail" },
                  { key: "org", label: "Organization" },
                  { key: "actions", label: "" },
                ]}
                scopedSlots={{
                  actions: (item) => (
                    <td>
                      <CButton
                        color="primary"
                        onClick={(e) => dispatch(deleteUser(item.thumbPrint))}
                      >
                        Delete
                      </CButton>
                    </td>
                  ),
                }}
                hover
                striped
              />
            </CCardBody>
          </CCard>
        </CCol>
        <CCol xl={6}>
          <CCard>
            <CCardHeader>Authorize new user</CCardHeader>
            <CCardBody>
              <CForm>
                <CFormGroup row>
                  <CCol xs={3}>
                    <CLabel htmlFor="certFile">User certificate:</CLabel>
                  </CCol>
                  <CCol xs={9}>
                    <CInputFile
                      onChange={(e) => {
                        const reader = new FileReader();

                        reader.onload = async (event) => {
                          setNewUser(event.target.result);
                        };
                        reader.onerror = (err) => {
                          console.log("Got onerror event", err);
                        };

                        reader.readAsArrayBuffer(e.target.files[0]);
                      }}
                    />
                  </CCol>
                </CFormGroup>
                <CFormGroup row>
                  <CCol xs={{ size: 4, offset: 4 }}>
                    <CButton
                      color="primary"
                      block
                      onClick={(e) => {
                        dispatch(addUser(newUser));
                        setNewUser(null);
                      }}
                    >
                      Authorize
                    </CButton>
                  </CCol>
                </CFormGroup>
              </CForm>
            </CCardBody>
          </CCard>
        </CCol>
      </CRow>
    </CContainer>
  );
};

export default Config;

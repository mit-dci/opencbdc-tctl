import Moment from "react-moment";
import {
  CCardFooter,
  CButton,
  CCol,
  CRow,
  CCard,
  CCardHeader,
  CCardBody,
  CFormGroup,
  CLabel,
  CInput,
  CSelect,
  CInputCheckbox,
  CTextarea,
} from "@coreui/react";
import { useDispatch, useSelector } from "react-redux";
import {
  deleteSavedPlot,
  generateSweepPlot,
  loadSavedPlots,
  enrichPlotConfig,
} from "../state/slices/testruns";
import { _ } from "core-js";
import { TestController } from "../state/actions";
import { useState } from "react";
import client from "../state/apiclient";

const SweepPlot = (props) => {
  const [config, setConfig] = useState({
    pointStyle: "^",
    pointStyle2: "s",
    sweepID: props.match.params.sweepID,
    xMax: -1,
    xMin: -1,
    yMax: -1,
    yMin: -1,
    x2Max: -1,
    x2Min: -1,
    y2Max: -1,
    y2Min: -1,
    successThreshold: 80,
  });
  const sweepPlotState = useSelector((s) => s.testruns.sweepPlots);
  const plotTypes = sweepPlotState.types;
  const plotFields = sweepPlotState.fields;
  const savedPlots = useSelector((s) =>
    s.testruns.savedSweepPlots.filter(
      (p) => p.sweepID === props.match.params.sweepID
    )
  );
  const [showRaw, setShowRaw] = useState(false);
  const plot = useSelector((s) => s.testruns.sweepPlot);
  const scales = [
	{ id: "linear", name: "Linear" },
	{ id: "log", name: "Logarithmic" }
  ] 
  const pointStyles = [
    { id: ".", name: "point" },
    { id: ",", name: "pixel" },
    { id: "o", name: "circle" },
    { id: "v", name: "triangle_down" },
    { id: "^", name: "triangle_up" },
    { id: "<", name: "triangle_left" },
    { id: ">", name: "triangle_right" },
    { id: "1", name: "tri_down" },
    { id: "2", name: "tri_up" },
    { id: "3", name: "tri_left" },
    { id: "4", name: "tri_right" },
    { id: "8", name: "octagon" },
    { id: "s", name: "square" },
    { id: "p", name: "pentagon" },
    { id: "P", name: "plus (filled)" },
    { id: "*", name: "star" },
    { id: "h", name: "hexagon1" },
    { id: "H", name: "hexagon2" },
    { id: "+", name: "plus" },
    { id: "x", name: "x" },
    { id: "X", name: "x (filled)" },
    { id: "D", name: "diamond" },
    { id: "d", name: "thin_diamond" },
    { id: "|", name: "vline" },
    { id: "_", name: "hline" },
  ];
  const dispatch = useDispatch();
  dispatch(loadSavedPlots(props.match.params.sweepID));
  const axes = useSelector((s) => s.testruns.sweepPlots.axes);
  return (
    <>
      <CRow>
        <CCol>
          <h1>Generate plot for sweep {props.match.params.sweepID}</h1>
        </CCol>
      </CRow>

      <CRow>
        <CCol xs={6}>
          <CCard>
            <CCardHeader>
              <b>Parameters</b>
            </CCardHeader>
            <CCardBody>
              <CFormGroup row>
                <CCol xs={4}>
                  <CLabel xs={6}>Title:</CLabel>
                </CCol>
                <CCol xs={8}>
                  <CInput
                    type="text"
                    value={config.title}
                    onChange={(e) => {
                      setConfig(
                        Object.assign({}, config, { title: e.target.value })
                      );
                    }}
                  ></CInput>
                </CCol>
              </CFormGroup>
              <CFormGroup row>
                <CCol xs={4}>
                  <CLabel xs={6}>Type:</CLabel>
                </CCol>
                <CCol xs={8}>
                  <CSelect
                    type="text"
                    value={config.type}
                    onChange={(e) => {
                      setConfig(
                        Object.assign({}, config, { type: e.target.value })
                      );
                    }}
                  >
                    <option value="" id="null">
                      -- select --
                    </option>
                    {plotTypes.map((t) => (
                      <option value={t.id} key={t.id}>
                        {t.name}
                      </option>
                    ))}
                  </CSelect>
                </CCol>
              </CFormGroup>
              <CFormGroup row>
                    <CCol xs={4}>
                      <CLabel xs={6}>Create series from field:</CLabel>
                    </CCol>
                    <CCol xs={8}>
                      <CSelect
                        type="text"
                        value={config[`seriesField`]}
                        onChange={(e) => {
                          let obj = {};
                          obj[`seriesField`] = e.target.value;
                          setConfig(Object.assign({}, config, obj));
                        }}
                      >
                        <option value="" id="null">
                          -- none --
                        </option>
                        {plotFields.map((t) => (
                          <option value={t.id} key={t.id}>
                            {t.name}
                          </option>
                        ))}
                      </CSelect>
                    </CCol>
                  </CFormGroup>
              {axes.map((ax) => (
                <>
                  <CFormGroup row>
                    <CCol xs={4}>
                      <CLabel xs={6}>{ax.name} field:</CLabel>
                    </CCol>
                    <CCol xs={8}>
                      <CSelect
                        type="text"
                        value={config[`${ax.id}Field`]}
                        onChange={(e) => {
                          let obj = {};
                          obj[`${ax.id}Field`] = e.target.value;
                          setConfig(Object.assign({}, config, obj));
                        }}
                      >
                        <option value="" id="null">
                          -- none --
                        </option>
                        {plotFields.map((t) => (
                          <option value={t.id} key={t.id}>
                            {t.name}
                          </option>
                        ))}
                      </CSelect>
                    </CCol>
                  </CFormGroup>
                <CFormGroup row>
                    <CCol xs={4}>
                      <CLabel xs={6}>{ax.name} scale:</CLabel>
                    </CCol>
                    <CCol xs={8}>
                      <CSelect
                        type="text"
                        value={config[`${ax.id}Scale`]}
                        onChange={(e) => {
                          let obj = {};
                          obj[`${ax.id}Scale`] = e.target.value;
                          setConfig(Object.assign({}, config, obj));
                        }}
                      >
                        {scales.map((t) => (
                          <option value={t.id} key={t.id}>
                            {t.name}
                          </option>
                        ))}
                      </CSelect>
                    </CCol>
                  </CFormGroup>
                  <CFormGroup row>
                    <CCol xs={4}>
                      <CLabel xs={6}>{ax.name} Min/Max (-1 = None):</CLabel>
                    </CCol>
                    <CCol xs={4}>
                      <CInput
                        type="text"
                        value={
                          config[`${ax.id}Min`] === undefined
                            ? -1
                            : config[`${ax.id}Min`]
                        }
                        onChange={(e) => {
                          let val = parseInt(e.target.value);
                          if (Number.isNaN(val)) {
                            val = 0;
                          }
                          let obj = {};
                          obj[`${ax.id}Min`] = val;
                          setConfig(Object.assign({}, config, obj));
                        }}
                      ></CInput>
                    </CCol>
                    <CCol xs={4}>
                      <CInput
                        type="text"
                        value={
                          config[`${ax.id}Max`] === undefined
                            ? -1
                            : config[`${ax.id}Max`]
                        }
                        onChange={(e) => {
                          let val = parseInt(e.target.value);
                          if (Number.isNaN(val)) {
                            val = 0;
                          }
                          let obj = {};
                          obj[`${ax.id}Max`] = val;
                          setConfig(Object.assign({}, config, obj));
                        }}
                      ></CInput>
                    </CCol>
                  </CFormGroup>
                </>
              ))}
              <CFormGroup row>
              <CCol xs={4}>
                  <CLabel xs={6}>Group By X value:</CLabel>
                </CCol>
                <CCol xs={8}>
              <CInputCheckbox
                    checked={config.groupByX === true}
                    onChange={(e) => {
                      setConfig(
                        Object.assign({}, config, { groupByX: e.target.checked, groupByXEval: 'max(yVals)'})
                      );
                    }}
                  />
                  </CCol>
              </CFormGroup>
              {config.groupByX === true && <CFormGroup row>
                    <CCol xs={4}>
                      <CLabel xs={6}>Grouping function:</CLabel>
                    </CCol>
                    <CCol xs={8}>
                      <CSelect
                        type="text"
                        value={config[`groupByXEval`]}
                        onChange={(e) => {
                          let cfg = Object.assign({}, config, {groupByXEval:e.target.value});
                          setConfig(cfg);
                        }}
                      >
                        <option value="max(yVals)">
                          Maximum
                        </option>
                        <option value="min(yVals)">
                          Minimum
                        </option>
                        <option value="avg(yVals)">
                          Average
                        </option>
                      </CSelect>
                    </CCol>
                  </CFormGroup>}
              <CFormGroup row>
                <CCol xs={4}>
                  <CLabel xs={6}>Success threshold (%):</CLabel>
                  <br />
                  <CLabel>
                    <small>
                      If less than this percentage of runs were succesful for
                      the given sweep value, that sweep value is treated as
                      having no results.
                    </small>
                  </CLabel>
                </CCol>
                <CCol xs={8}>
                  <CInput
                    type="text"
                    value={
                      config[`successThreshold`] === undefined
                        ? 80
                        : config[`successThreshold`]
                    }
                    onChange={(e) => {
                      let val = parseInt(e.target.value);
                      if (Number.isNaN(val)) {
                        val = 0;
                      }
                      let obj = {};
                      obj[`successThreshold`] = val;
                      setConfig(Object.assign({}, config, obj));
                    }}
                  ></CInput>
                </CCol>
              </CFormGroup>
              <CFormGroup row>
                <CCol xs={4}>
                  <CLabel xs={6}>Point style 1:</CLabel>
                </CCol>
                <CCol xs={8}>
                  <CSelect
                    type="text"
                    value={config.pointStyle}
                    onChange={(e) => {
                      let obj = {};
                      obj[`pointStyle`] = e.target.value;
                      setConfig(Object.assign({}, config, obj));
                    }}
                  >
                    {pointStyles.map((t) => (
                      <option value={t.id} key={t.id}>
                        {t.name}
                      </option>
                    ))}
                  </CSelect>
                </CCol>
              </CFormGroup>
              <CFormGroup row>
                <CCol xs={4}>
                  <CLabel xs={6}>Point style 2:</CLabel>
                </CCol>
                <CCol xs={8}>
                  <CSelect
                    type="text"
                    value={config.pointStyle2}
                    onChange={(e) => {
                      let obj = {};
                      obj[`pointStyle2`] = e.target.value;
                      setConfig(Object.assign({}, config, obj));
                    }}
                  >
                    {pointStyles.map((t) => (
                      <option value={t.id} key={t.id}>
                        {t.name}
                      </option>
                    ))}
                  </CSelect>
                </CCol>
              </CFormGroup>
              <CRow>
                <CCol xs={{ size: 4, offset: 4 }}>
                  <center>
                    <CButton
                      color="primary"
                      onClick={(e) => {
                        dispatch(generateSweepPlot(config));
                      }}
                    >
                      Create
                    </CButton>
                  </center>
                </CCol>
              </CRow>
              <CFormGroup row>
                <CCol xs={12} style={{ textAlign: "center" }}>
                  <CInputCheckbox
                    checked={config.save === true}
                    onChange={(e) => {
                      setConfig(
                        Object.assign({}, config, { save: e.target.checked })
                      );
                    }}
                  />
                  {" Save plot"}
                </CCol>
              </CFormGroup>
              <CRow>
                <CCol xs={{ size: 4, offset: 4 }}>
                  <center>
                    <CButton
                      color="primary"
                      onClick={(e) => {
                        setShowRaw(!showRaw)
                      }}
                    >
                      {showRaw ? 'Hide' : 'Show'}{' raw definition'}
                    </CButton>
                  </center>
                </CCol>
              </CRow>
              {showRaw === true && <CFormGroup row>
                <CCol xs={12} style={{ textAlign: "center" }}>
                  <CTextarea rows={10} value={JSON.stringify(enrichPlotConfig(config,sweepPlotState))} readOnly={true}/>
                </CCol>
              </CFormGroup>}
            </CCardBody>
          </CCard>
        </CCol>
        {plot && (
          <CCol xs={6}>
            <CCard>
              <CCardHeader>
                <b>Plot</b>
              </CCardHeader>
              <CCardBody>
                <img style={{ width: "100%" }} src={plot} />
              </CCardBody>
            </CCard>
          </CCol>
        )}
      </CRow>
      {savedPlots && savedPlots.length > 0 && (
        <CRow>
          <CCol xs={12}>
            <h2>Saved plots</h2>
            <CRow>
              {savedPlots.map((sp) => (
                <CCol xs={3}>
                  <CCard>
                    <CCardHeader>
                      <b>{sp.title}</b>
                      <br />
                      <small>
                        <Moment format="LL">{sp.date}</Moment>
                      </small>
                    </CCardHeader>
                    <CCardBody>
                      <img
                        src={`${client.apiUrl}sweepplot/saved/${sp.sweepID}/${sp.id}`}
                        style={{ cursor: "pointer", width: "100%" }}
                        onClick={(e) => {
                          setConfig(
                            Object.assign({}, sp.request, { save: null })
                          );
                          dispatch({
                            type: TestController.SweepPlotGenerated,
                            payload: {
                              plot: `${client.apiUrl}sweepplot/saved/${sp.sweepID}/${sp.id}`,
                            },
                          });
                        }}
                      />
                    </CCardBody>
                    <CCardFooter>
                      <CButton
                        style={{ float: "right" }}
                        color="danger"
                        onClick={(e) => {
                          dispatch(deleteSavedPlot(sp.sweepID, sp.id));
                        }}
                      >
                        Delete
                      </CButton>
                    </CCardFooter>
                  </CCard>
                </CCol>
              ))}
            </CRow>
          </CCol>
        </CRow>
      )}
    </>
  );
};

export default SweepPlot;

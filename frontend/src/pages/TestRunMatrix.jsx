import { CButton } from '@coreui/react';
import { useHistory } from 'react-router-dom';
import { useDispatch, useSelector } from 'react-redux';
import { loadTestRunSweepMatrix} from '../state/slices/testruns';
import './TestRunMatrix.css';
import { _ } from 'core-js';
import client from '../state/apiclient';

const TestRunMatrix = (props) => {
  const history = useHistory();
  const sweepMatrix = useSelector(state => state.testruns.sweepMatrix.sort((a,b) => {
    return a.config.architectureID.localeCompare(b.config.architectureID) || a.config.atomizers-b.config.atomizers || a.config.coordinators-b.config.coordinators || a.config.sentinels-b.config.sentinels || a.config.shards-b.config.shards || a.config.clients-b.config.clients || a.config.watchtowers-b.config.watchtowers || a.config.batchSize-b.config.batchSize || a.config.windowSize-b.config.windowSize
  }));
  const dispatch = useDispatch();
  dispatch(loadTestRunSweepMatrix(props.match.params.sweepID));
  let matrix = sweepMatrix;

  const metrics = ['Throughput','Latency'];
  const metricsDetails = ['Avg','Min','Max','Std'];
  if(!matrix || matrix.length === 0) {
    return <h1>Loading...</h1>
  }

  return <><h1>Result matrix</h1>
  <CButton color="primary"  style={{float:'right'}} onClick={(e) => { window.open(`${client.apiUrl}testruns/${props.match.params.sweepID ? 'sweepMatrixCsv/' + props.match.params.sweepID : 'matrixcsv'}`)}}>Download CSV</CButton>
    <div style={{width: '100%', height: '100%', overflow: 'auto'}}><table className="testrunmatrix">
      <thead>
        <tr>
          {Object.keys(matrix[0].config).map(k => <th className="rotate">
            <div><span>{k}</span></div>
          </th>)}
          <th className="rotate">
            <div><span>Result count</span></div>
          </th>
          {metrics.map(m => {

            return <>
                  {metricsDetails.map(md => <th className="rotate">
                      <div><span>{m} {md}</span></div>
                    </th>
                  )}
                  {matrix[0].result[m.toLowerCase() + 'Percentiles'].map(p => <th className="rotate">
                    <div><span>{m} {p.bucket}</span></div>
                  </th>)}
            </>

          })}
        </tr>
      </thead>
      <tbody>
        {matrix.map(m => {
          return <tr>{Object.keys(m.config).map(k => {
            if (m.config[k] === true || m.config[k] === false) {
              return <td>{m.config[k] === true ? "Yes" : "No"}</td>;
            } else {
              return <td>{m.config[k]}</td>;
            }
          })}
          <td>{m.resultCount}</td>
          {metrics.map(mtrc => {
            return <>
                  {metricsDetails.map(md => <td>
                      {mtrc === "Latency" ? (Math.floor(m.result[`${mtrc.toLowerCase()}${md}`] * 10000) / 10).toString() + "ms" : Math.floor(m.result[`${mtrc.toLowerCase()}${md}`]).toString()}
                    </td>
                  )}
                  {m.result[mtrc.toLowerCase() + 'Percentiles'].map(p => <td>
                    {mtrc === "Latency" ? `${Math.floor(p.value * 10000)/10}ms` : `${Math.floor(p.value)}`}
                  </td>)}
            </>

            })}
          </tr>
        })}
      </tbody>
    </table></div></>
};

export default TestRunMatrix;

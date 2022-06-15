import { connect } from '@giantmachines/redux-websocket';
import client from '../apiclient';
import { TestController, ReduxWebSocket } from '../actions';
import { toast } from 'react-toastify';
import { loadTestRunDetails } from '../slices/testruns';

export const loadWebsocketToken = async (dispatch, getState) => {
    try {
        if (!getState().system?.websocketState?.connecting) {
            dispatch({ type: TestController.WebsocketStateChanged, payload: { connecting: true } });
            let token = await client.get("wsToken");
            dispatch({ type: TestController.WebsocketTokenReceived, payload: token });
        }
    } catch (e) {
        dispatch({ type: TestController.WebsocketStateChanged, payload: { connecting: false } });
        setTimeout(() => {
            dispatch(loadWebsocketToken);
        }, 2000);
    }
}

const websocketMiddleware = storeAPI => next => action => {
    switch (action.type) {
        case TestController.WebsocketTokenReceived:
            storeAPI.dispatch(connect(action.payload.target));
            storeAPI.dispatch({ type: TestController.WebsocketStateChanged, payload: { connecting: false } });
            break;
        case ReduxWebSocket.Open:
            storeAPI.dispatch({ type: TestController.Toast.Success, payload: "Connected to web socket" });
            storeAPI.dispatch({ type: TestController.WebsocketStateChanged, payload: { connected: true, connecting: false } });
            break;
        case ReduxWebSocket.Closed:
            storeAPI.dispatch({ type: TestController.Toast.Error, payload: "Disconnected from web socket, reconnecting" });
            storeAPI.dispatch({ type: TestController.WebsocketStateChanged, payload: { connected: false } });
            setTimeout(() => {
                storeAPI.dispatch(loadWebsocketToken);
            }, 2000);
            break;
        case ReduxWebSocket.Message:
            let msg = action.payload.message;
            switch (msg.type) {
                case "maintenanceModeChanged":
                    storeAPI.dispatch({ type: TestController.MaintenanceModeChanged, payload: msg.payload })
                    break;
                case "testRunManagerConfigUpdated":
                    storeAPI.dispatch({ type: TestController.ConfigChanged, payload: msg.payload })
                    break;
                case "agentCountChanged":
                    storeAPI.dispatch({ type: TestController.AgentCountUpdated, payload: {count:msg.payload.count}})
                case "testRunCreated":
                    storeAPI.dispatch({ type: TestController.TestRunAdded, payload: msg.payload.data })
                    break;
                case "systemStateChange":
                    storeAPI.dispatch({ type: TestController.SystemStateChanged, payload: msg.payload })
                    break;
                case "testRunStatusChanged":
                    storeAPI.dispatch({
                        type: TestController.TestRunChanged, payload: {
                            id: msg.payload.testRunID,
                            status: msg.payload.status,
                            started: msg.payload.started,
                            completed: msg.payload.completed,
                            details: msg.payload.details
                        }
                    })
                    if (msg.payload.status === "Completed") {
                        toast.success(`Test run ${msg.payload.testRunID} completed`);
                        // Trigger reloading all details from the server to fetch the
                        // completed commands.
                        storeAPI.dispatch(loadTestRunDetails(msg.payload.testRunID));
                    }
                    if (msg.payload.status === "Failed") {
                        toast.error(`Test run ${msg.payload.testRunID} failed : ${msg.payload.details}`);
                    }
                    break;
                case "testRunTrimParametersChange":
                    storeAPI.dispatch({
                        type: TestController.TestRunChanged, payload: {
                            id: msg.payload.testRunID,
                            trimZeroesAtStart: msg.payload.trimZeroes,
                            trimZeroesAtEnd: msg.payload.trimZeroesEnd,
                            trimSamplesAtStart: msg.payload.trimSamples,
                        }
                    })
                    break;
                case "testRunRolesChanged":
                    storeAPI.dispatch({
                        type: TestController.TestRunChanged, payload: {
                            id: msg.payload.testRunID,
                            roles: msg.payload.roles,
                        }
                    });
                    break;
                case "testRunResultAvailable":
                    let avgThroughput = -1;
                    if(msg.payload.result?.throughputAvg) {
                        avgThroughput = msg.payload.result?.throughputAvg
                    }

                    let tailLatency = -1;
                    if(msg.payload.result?.latencyPercentiles) {
                        let ninetyNine = msg.payload.result?.latencyPercentiles.find(p => p.bucket === 99)
                        if(ninetyNine) {
                            tailLatency = ninetyNine.value;
                        }
                    }

                    storeAPI.dispatch({
                        type: TestController.TestRunChanged, payload: {
                            id: msg.payload.testRunID,
                            result: msg.payload.result,
                            avgThroughput: avgThroughput,
                            tailLatency: tailLatency,
                            resultUpdated: new Date(),
                        }
                    });
                    break;
                case "testRunLogAppended":
                    storeAPI.dispatch({ type: TestController.TestRunLogAppended, payload: msg.payload });
                    break;
                case "redownloadComplete":
                    if(msg.payload.success) {
                        storeAPI.dispatch({ type: TestController.Toast.Success, payload: `Redownload of testrun ${msg.payload.testRunID} outputs succeeded` });
                    } else {
                        storeAPI.dispatch({ type: TestController.Toast.Error, payload: `Redownload of testrun ${msg.payload.testRunID} outputs failed: ${msg.payload.error}` });
                    }
                    break;
                case "testRunExecutedCommandAdded":
                    storeAPI.dispatch({
                        type: TestController.TestRunExecutedCommandAdded, payload: msg.payload
                    })
                    break;
                case "connectedUsersChanged":
                    storeAPI.dispatch({ type: TestController.OnlineUsersChanged, payload: msg.payload.count })
                    break;
            }
            break;
    }
    // Do something in here, when each action is dispatched
    return next(action)
}

export default websocketMiddleware;
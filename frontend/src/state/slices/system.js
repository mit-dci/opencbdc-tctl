import {TestController, ReduxWebSocket} from '../actions';
import { createSelector } from 'reselect';
import client from '../apiclient';

export const reducer = (state = {initialStateLoaded: false, systemState: "unknown", maintenanceMode:false, version:'', websocketState:{connected:false, connecting:false}}, action) => {
    switch(action.type) {
        case TestController.InitialStateLoaded:
            return {...state, initialStateLoaded: true, config:action.payload?.config || {maxAgents:500}, maintenanceMode:action.payload?.maintenance || false, webSocketConn: 0, version:action.payload?.version || '', onlineUsers: action.payload?.onlineUsers || 0};
        case TestController.OnlineUsersChanged:
            return {
                ...state,
                onlineUsers: action.payload
            }
        case TestController.WebsocketStateChanged:
            return {
                ...state,
                websocketState: Object.assign({}, state.websocketState, action.payload)
            }
        case ReduxWebSocket.Connect:
            return {
                ...state,
                webSocketConn: state.webSocketConn + 1
            }
        case TestController.MaintenanceModeChanged:
            return {
                ...state,
                maintenanceMode:action.payload?.maintenanceMode
            }
        case TestController.ConfigChanged:
            return {
                ...state,
                config:action.payload?.config
            }
        case TestController.SystemStateChanged:
            return {
                ...state,
                systemState:action.payload?.state
            }
        default:
            return state;
    }
}

export const toggleMaintenanceMode = async () => {
    await client.put("maintenance");
}

export const setMaxAgents = async (max) => {
    await client.put(`testruns/maxagents/${max}`);
}
import {TestController} from '../actions';

export const reducer = (state = {agentCount:0, launchTemplates:[]}, action) => {
    switch(action.type) {
        case TestController.InitialStateLoaded:
            return {agentCount:(action.payload?.agentCount || 0), launchTemplates:(action.payload?.launchTemplates || [])};
        case TestController.AgentCountUpdated:
            return {
                ...state,
                agentCount: action.payload.count
            }
        default:
            return state;
    }
}
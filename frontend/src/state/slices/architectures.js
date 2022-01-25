import {TestController} from '../actions';
import { createSelector } from 'reselect'

export const reducer = (state = {architectures:[]}, action) => {
    switch(action.type) {
        case TestController.InitialStateLoaded:
            return {architectures:action.payload?.architectures || [], roleCompositions:action.payload?.roleCompositions || []};
        default:
            return state;
    }
}
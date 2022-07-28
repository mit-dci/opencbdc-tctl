import { TestController } from '../actions';
import { createSelector } from 'reselect'
import { send } from '@giantmachines/redux-websocket/dist';
import client from '../apiclient';
import * as numeral from "numeral";

const initialState = {
    testruns: [],
    testrunFields: [],
    testrunLogs: [],
    architectures: [],
    activeCommandLog: {
        id: '',
        log: '',
        view: '',
    },
    newestTestRunIDWhenMatrixFetched: '',
    matrix: [],
    sweeps: [],
    sweepMatrix: [],
    sweepMatrixSweepID: '',
    sweepPlot: null,
    savedSweepPlots: [],
    savedSweepPlotsSweepID: '',
    sweepPlotParams: { type: 'line' },
    sweepPlots: {},
    scheduledTestRun: {roles:[], sweepRoles:[]}
};

export const reducer = (state = initialState, action) => {
    switch (action.type) {
        case TestController.InitialStateLoaded:
            return {
                ...state,
                architectures: action.payload?.architectures || [],
                perfGraphs: action.payload?.perfGraphs || [],
                testruns: action.payload?.testruns || [],
                testrunFields: action.payload?.testRunFields || [],
                sweeps: action.payload?.sweeps || [],
                sweepPlots: action.payload?.sweepPlotConfig || {},
                scheduledTestRun: action.payload?.architectures[0].defaultTest || {},
            };
        case TestController.CommandLogDownloaded:
            return {
                ...state,
                activeCommandLog: action.payload
            }
        case TestController.ScheduleTestRun.Reschedule:
            var trun = state.testruns.find(
                (tr) => tr.id === action.payload.id
            );
            if (trun) {
                return {
                    ...state,
                    scheduledTestRun: Object.assign({}, trun, {
                        roles: trun.roles.map((rl) => {
                            var agentChoiceId = "";
                            if (rl.awsLaunchTemplateID !== "") {
                                agentChoiceId = `AWS-${rl.awsLaunchTemplateID}`;
                            }
                            return Object.assign({}, rl, { agentId: -1 })
                        }),
                    })
                }
            } else {
                return state;
            }
        case TestController.SweepPlotsLoaded:
            {
                return {
                    ...state,
                    savedSweepPlotsSweepID: action.payload.sweepID,
                    savedSweepPlots: action.payload.sweepPlots.map(sp => Object.assign({}, sp, { sweepID: action.payload.sweepID }))
                }
            }
        case TestController.SweepPlotGenerated:
            return {
                ...state,
                sweepPlot: action.payload.plot,
            }
        case TestController.TestRunAdded:
            if(!action.payload) {
                return state;
            }
            return {
                ...state,
                testruns: [
                    ...state.testruns.filter((tr) => tr !== undefined),
                    action.payload
                ]
            };
        case TestController.TestRunChanged:
            return {
                ...state,
                testruns: state.testruns.filter((tr) => tr !== undefined).map((tr) => {
                    if (tr.id !== action.payload.id) return tr;
                    return Object.assign({}, tr, action.payload);
                })
            }
        case TestController.TestRunMatrixUpdated:
            return {
                ...state,
                matrix: action.payload.matrix,
                newestTestRunIDWhenMatrixFetched: action.payload.newestRun
            }
        case TestController.TestRunSweepMatrixUpdated:
            return {
                ...state,
                sweepMatrix: action.payload.matrix,
                sweepMatrixSweepID: action.payload.sweepID,
            }
        case TestController.ScheduleTestRun.ChangeProperty:
            let newRun = Object.assign({}, state.scheduledTestRun, action.payload);
            if(action.payload.architectureID && action.payload.architectureID !== state.scheduledTestRun.architectureID) {
                newRun = Object.assign({}, state.architectures.find(a => a.id === action.payload.architectureID).defaultTest);
            }
            return {
                ...state,
                scheduledTestRun: newRun
            }
        case TestController.ScheduleTestRun.ApplyRoleFailure:
            {
                let newRoles = [];
                for (let role of state.scheduledTestRun.roles) {
                    let fail = false
                    let failure = action.payload.failure.after == -1 ? null : { after: action.payload.failure.after };

                    for (let failSubject of action.payload.failure.what) {
                        if (role.role === failSubject.role && role.roleIdx === failSubject.index) {
                            fail = true;
                            break;
                        }
                    }
                    if (fail) {
                        newRoles.push(Object.assign({}, role, { failure }));
                    } else {
                        newRoles.push(role);
                    }
                }
                return {
                    ...state,
                    scheduledTestRun: {
                        ...state.scheduledTestRun,
                        roles: newRoles
                    }
                }
            }
        case TestController.ScheduleTestRun.ApplyRoleConfig:
            if (action.payload.config.role === '') return state;
            let newRoles = [];
            for (let role of state.scheduledTestRun.roles) {
                if (role.role !== action.payload.config.role) {
                    newRoles.push(role);
                }
            }
            for (let i = 0; i < action.payload.config.count; i++) {
                let agentChoice = action.payload.config.agentChoices[i % action.payload.config.agentChoices.length];
                newRoles.push({
                    role: action.payload.config.role,
                    roleIdx: i,
                    agentID: -1,
                    agentChoiceId: `AWS-${agentChoice.id}`,
                    awsLaunchTemplateID: agentChoice.id,
                });
            }
            return {
                ...state,
                scheduledTestRun: {
                    ...state.scheduledTestRun,
                    roles: newRoles
                }
            }
        case TestController.ScheduleTestRun.ApplySweepRoleConfig:
            if (action.payload.config.role === '') return state;
            let newSweepRoles = [];
            for (let role of state.scheduledTestRun.sweepRoles) {
                if (role.role !== action.payload.config.role) {
                    newSweepRoles.push(role);
                }
            }
            for (let i = 0; i < action.payload.config.count; i++) {
                let agentChoice = action.payload.config.agentChoices[i % action.payload.config.agentChoices.length];
                newSweepRoles.push({
                    role: action.payload.config.role,
                    roleIdx: i,
                    agentID: -1,
                    agentChoiceId: `AWS-${agentChoice.id}`,
                    awsLaunchTemplateID: agentChoice.id,
                });
            }
            return {
                ...state,
                scheduledTestRun: {
                    ...state.scheduledTestRun,
                    sweepRoles: newSweepRoles
                }
            }
        case TestController.ScheduleTestRun.AddNewRole:
            {
                let newIdx = 0;
                let freeAgentId = -1;
                let newRoles = [];
                for (let role of state.scheduledTestRun.roles) {
                    if (role.role === action.payload) {
                        newIdx++;
                    }
                    newRoles.push(role);
                }
                let newRole = {
                    role: action.payload,
                    roleIdx: newIdx,
                    agentID: -1,
                    agentChoiceId: "",
                };
                newRoles.push(newRole);
                return {
                    ...state,
                    scheduledTestRun: {
                        ...state.scheduledTestRun,
                        roles: newRoles
                    }
                }
            }
        case TestController.ScheduleTestRun.DeleteRole:
            {
                let newRoles = [];
                var removed = false;
                for (var j = state.scheduledTestRun.roles.length - 1; j >= 0; j--) {
                    if (state.scheduledTestRun.roles[j].role === action.payload && !removed) {
                        removed = true
                    } else {
                        newRoles.unshift(state.scheduledTestRun.roles[j]);
                    }
                }
                return {
                    ...state,
                    scheduledTestRun: {
                        ...state.scheduledTestRun,
                        roles: newRoles
                    }
                }
            }
        case TestController.ScheduleTestRun.AddNewSweepRole:
            {
                let newIdx = 0;
                let freeAgentId = -1;
                let newRoles = [];
                for (let role of state.scheduledTestRun.sweepRoles) {
                    if (role.role === action.payload) {
                        newIdx++;
                    }
                    newRoles.push(role);
                }
                let newRole = {
                    role: action.payload,
                    roleIdx: newIdx,
                    agentID: -1,
                    agentChoiceId: "",
                };
                newRoles.push(newRole);
                return {
                    ...state,
                    scheduledTestRun: {
                        ...state.scheduledTestRun,
                        sweepRoles: newRoles
                    }
                }
            }
        case TestController.ScheduleTestRun.DeleteSweepRole:
            {
                let newRoles = [];
                var removed = false;
                for (var j = state.scheduledTestRun.sweepRoles.length - 1; j >= 0; j--) {
                    if (state.scheduledTestRun.sweepRoles[j].role === action.payload && !removed) {
                        removed = true
                    } else {
                        newRoles.unshift(state.scheduledTestRun.sweepRoles[j]);
                    }
                }
                return {
                    ...state,
                    scheduledTestRun: {
                        ...state.scheduledTestRun,
                        sweepRoles: newRoles
                    }
                }
            }
        case TestController.ScheduleTestRun.ApplyRoleComposition:
            {
                let newRoles = [];
                let roleIdxs = {};
                for (var role of action.payload.roles) {
                    if (roleIdxs[role] !== undefined) {
                        roleIdxs[role] = roleIdxs[role] + 1;
                    } else {
                        roleIdxs[role] = 0;
                    }
                    newRoles.push({
                        role: role,
                        roleIdx: roleIdxs[role],
                        agentID: -1,
                        agentChoiceId: "",
                    });
                }
                return {
                    ...state,
                    scheduledTestRun: {
                        ...state.scheduledTestRun,
                        roles: newRoles
                    }
                }
            }
        case TestController.ScheduleTestRun.AssignSweepRoleAgent:
            {
                let newRoles = [];
                for (let role of state.scheduledTestRun.sweepRoles) {
                    if (role.role === action.payload.roleToSet && role.roleIdx === action.payload.roleIdx) {
                        let agentProps = { agentID: -1 };
                        agentProps.awsLaunchTemplateID = action.payload.choice.substring(4);
                        newRoles.push(
                            Object.assign({}, role, agentProps, { agentChoiceId: action.payload.choice })
                        );
                    } else {
                        newRoles.push(role);
                    }
                }
                return {
                    ...state,
                    scheduledTestRun: {
                        ...state.scheduledTestRun,
                        sweepRoles: newRoles
                    }
                }
            }
        case TestController.ScheduleTestRun.AssignRoleAgent:
            {
                let newRoles = [];
                for (let role of state.scheduledTestRun.roles) {
                    if (role.role === action.payload.roleToSet && role.roleIdx === action.payload.roleIdx) {
                        let agentProps = { agentID: -1 };
                        if (action.payload.choice.startsWith("AWS-")) {
                            agentProps.awsLaunchTemplateID = action.payload.choice.substring(4);
                        } else {
                            agentProps.awsLaunchTemplateID = "";
                            agentProps.agentID = parseInt(action.payload.choice.substring(4));
                        }

                        newRoles.push(
                            Object.assign({}, role, agentProps, { agentChoiceId: action.payload.choice })
                        );
                    } else {
                        newRoles.push(role);
                    }
                }
                return {
                    ...state,
                    scheduledTestRun: {
                        ...state.scheduledTestRun,
                        roles: newRoles
                    }
                }
            }
        case TestController.ScheduleTestRun.SetRoleFail:
            {
                let newRoles = [];
                for (let role of state.scheduledTestRun.roles) {
                    if (role.role === action.payload.role && role.roleIdx === action.payload.roleIdx) {
                        newRoles.push(
                            Object.assign({}, role, { fail: action.payload.fail })
                        );
                    } else {
                        newRoles.push(role);
                    }
                }
                return {
                    ...state,
                    scheduledTestRun: {
                        ...state.scheduledTestRun,
                        roles: newRoles
                    }
                }
            }
        case TestController.TestRunExecutedCommandAdded:
            return {
                ...state,
                testruns: state.testruns.map((tr) => {
                    if (tr.id !== action.payload.testRunID) return tr;
                    return {
                        ...tr,
                        executedCommands: [
                            ...(tr.executedCommands || []),
                            action.payload.details
                        ]
                    }
                })
            }
        case TestController.TestRunLogAppended:
            var oldLog = state.testrunLogs.find(tr => tr.id === action.payload.id);
            var newLog = action.payload;
            if (oldLog) {
                newLog.log = `${oldLog.log}${newLog.log}`;
            }

            return {
                ...state,
                testrunLogs: [
                    ...state.testrunLogs.filter(tr => tr.id !== action.payload.id),
                    newLog
                ]
            }
        default:
            return state;
    }
}

export const rescheduleTestRun = (id, sweep) => async (dispatch) => {
    try {
        let result = await client.get(`testruns/${id}/details`);
        if (result) {
            dispatch({ type: TestController.TestRunChanged, payload: Object.assign({}, result, { detailLoading: false, detailsAvailable: true }) });
            dispatch({ type: TestController.ScheduleTestRun.Reschedule, payload: { id, sweep } });
        }
    } catch (e) {
        dispatch({ type: TestController.Toast.Error, payload: e.message });
    }
}

export const continueOneAtATimeSweep = (id) => async (dispatch) => {
    try {
        let result = await client.get(`sweeps/${id}/continue`);
        if (result) {
            dispatch({ type: TestController.Toast.Success, payload: "Continued sweep scheduled" });
        } else {
            dispatch({ type: TestController.Toast.Error, payload: "Failed to continue sweep" });
        }
    } catch (e) {
        dispatch({ type: TestController.Toast.Error, payload: e.message });
    }
}


export const generateReport = (def) => async (dispatch) => {
    try {
        let result = await client.post('generateReport', def, false, true);
        let b = await result.blob();
        let tempLink = document.createElement('a');
        tempLink.href = URL.createObjectURL(b);
        tempLink.setAttribute('download', 'report.html');
        tempLink.click();
    } catch (e) {
        dispatch({ type: TestController.Toast.Error, payload: e.message });
    }
}

export const applyScheduledRunRoleFailure = (failure) => {
    let fl = {};
    if (failure.what === '') return { type: TestController.Toast.Error, payload: 'Select what to fail' };
    fl.what = failure.what;
    fl.after = parseInt(failure.after);
    if (Number.isNaN(fl.after)) return { type: TestController.Toast.Error, payload: 'Fail after is invalid' };
    return { type: TestController.ScheduleTestRun.ApplyRoleFailure, payload: { failure: fl } };
}

export const applyScheduledRunRoleConfig = (config) => {
    let cfg = {};
    if (config.role === '') return { type: TestController.Toast.Error, payload: 'Choose a role first' };
    cfg.role = config.role;
    cfg.count = parseInt(config.count);
    if (Number.isNaN(cfg.count)) return { type: TestController.Toast.Error, payload: 'Count is invalid' };
    cfg.agentChoices = config.agentChoices;
    return { type: TestController.ScheduleTestRun.ApplyRoleConfig, payload: { config: cfg } };
}

export const applySweepRunRoleConfig = (config) => {
    let cfg = {};
    if (config.role === '') return { type: TestController.Toast.Error, payload: 'Choose a role first' };
    cfg.role = config.role;
    cfg.count = parseInt(config.count);
    if (Number.isNaN(cfg.count)) return { type: TestController.Toast.Error, payload: 'Count is invalid' };
    cfg.agentChoices = config.agentChoices;
    return { type: TestController.ScheduleTestRun.ApplySweepRoleConfig, payload: { config: cfg } };
}

export const applyScheduledRunRoleComposition = (roles) => {
    return { type: TestController.ScheduleTestRun.ApplyRoleComposition, payload: { roles } };
}

export const setScheduledRunRoleAgent = (role, roleIdx, choice) => {
    return { type: TestController.ScheduleTestRun.AssignRoleAgent, payload: { roleToSet: role, roleIdx, choice } };
}

export const setScheduledRunProperty = prop => {
    return { type: TestController.ScheduleTestRun.ChangeProperty, payload: prop }
}

export const addScheduledRunRole = role => {
    return { type: TestController.ScheduleTestRun.AddNewRole, payload: role }
}

export const deleteScheduledRunRole = role => {
    return { type: TestController.ScheduleTestRun.DeleteRole, payload: role }
}

export const setScheduledRunRoleFail = (role, roleIdx, fail) => {
    return { type: TestController.ScheduleTestRun.SetRoleFail, payload: { role, roleIdx, fail } }
}

export const setScheduledRunSweepRoleAgent = (role, roleIdx, choice) => {
    return { type: TestController.ScheduleTestRun.AssignSweepRoleAgent, payload: { roleToSet: role, roleIdx, choice } };
}

export const addScheduledRunSweepRole = role => {
    return { type: TestController.ScheduleTestRun.AddNewSweepRole, payload: role }
}

export const deleteScheduledRunSweepRole = role => {
    return { type: TestController.ScheduleTestRun.DeleteSweepRole, payload: role }
}

export const downloadCommandLog = (id, view) => async (dispatch) => {
    try {
        let result = await client.call(`commands/${id}/output/${view}`, 'GET', undefined, false, true);
        const log = await result.text();
        dispatch({ type: TestController.CommandLogDownloaded, payload: {id,view,log}});
    } catch (e) {
        dispatch({ type: TestController.Toast.Error, payload: e.message });
    }
}

export const subscribeTestRunLog = id => {
    return send({ t: 'subscribeTestRunLog', m: { id } })
}

export const unsubscribeTestRunLog = () => {
    return send({ t: 'unsubscribeTestRunLog', m: {} })
}

export const validateAndScheduleTestRun = (history) => async (dispatch, getState) => {
    const testRun = getState().testruns.scheduledTestRun;
    let schedule = true;
    if(!testRun.commitHash) {
        dispatch({ type: TestController.Toast.Error, payload: "You have to select a commit before scheduling a test" });
        schedule = false;
    }
    if(!testRun.roles || testRun.roles.length === 0) {
        dispatch({ type: TestController.Toast.Error, payload: "You need to add at least one role to the system" });
        schedule = false;
    }
    if(schedule) {
        dispatch(scheduleTestRun(history));
    }
}

export const scheduleTestRun = (history) => async (dispatch, getState) => {
    try {
        const testRun = getState().testruns.scheduledTestRun;
        let result = await client.post("testruns/schedule", Object.assign({}, testRun, { invalidTxRate: parseFloat(testRun.invalidTxRate) }));
        if (result.ok === true) {
            dispatch({ type: TestController.Toast.Success, payload: "Test run scheduled successfully" });
            history.push("/testruns/running");
        } else {
            dispatch({ type: TestController.Toast.Error, payload: "Test run could not be scheduled, try again later" });
        }
    } catch (e) {
        dispatch({ type: TestController.Toast.Error, payload: e.message });
    }
}

export const estimateTestRun = (setResult) => async (dispatch, getState) => {
    try {
        const testRun = getState().testruns.scheduledTestRun;
        let result = await client.post("testruns/estimate", Object.assign({}, testRun, { invalidTxRate: parseFloat(testRun.invalidTxRate) }));
        if (result) {
            setResult(result);
        } else {
            dispatch({ type: TestController.Toast.Error, payload: "Test run could not be estimated, try again later" });
        }
    } catch (e) {
        dispatch({ type: TestController.Toast.Error, payload: e.message });
    }
}

export const rescheduleMissingSweepRuns = (id) => async (dispatch) => {
    try {
        let result = await client.get(`sweeps/${id}/fixMissing`);
        if (result.ok === true) {
            dispatch({ type: TestController.Toast.Success, payload: "Missing sweep runs scheduled successfully" });
        } else {
            dispatch({ type: TestController.Toast.Error, payload: "Missing sweep runs could not be scheduled, try again later" });
        }
    } catch (e) {
        dispatch({ type: TestController.Toast.Error, payload: e.message });
    }
}

export const enrichPlotConfig = (config, sweepPlotsState) => {
    let cfg = Object.assign({}, config);
    for (let axis of sweepPlotsState.axes) {
        let f = sweepPlotsState.fields.find(f => f.id === config[`${axis.id}Field`]);
        if (f) {
            cfg[`${axis.id}FieldName`] = f.name;
            cfg[`${axis.id}FieldNameShorthand`] = f.shortHand || f.name;
            cfg[`${axis.id}FieldEval`] = f.eval;
        }
    }
    let f = sweepPlotsState.fields.find(f => f.id === config[`seriesField`]);
    if (f) {
        cfg[`seriesFieldName`] = f.name;
        cfg[`seriesFieldNameShorthand`] = f.shortHand || f.name;
        cfg[`seriesFieldEval`] = f.eval;
    }
    return cfg;
}

export const generateSweepPlot = (config) => async (dispatch, getState) => {
    try {
        dispatch({ type: TestController.SweepPlotGenerated, payload: { plot: null } });
        let s = getState();
        let cfg = enrichPlotConfig(config, s.testruns.sweepPlots);
        let result = await client.post('sweepplot', cfg, false, true);
        let b = await result.blob();
        dispatch({ type: TestController.SweepPlotGenerated, payload: { plot: URL.createObjectURL(b) } });
        dispatch({ type: TestController.SweepPlotsLoaded, payload: { sweepID: '', sweepPlots: [] } });

    } catch (e) {
        dispatch({ type: TestController.Toast.Error, payload: e.message });
    }
}

export const loadSavedPlots = (sweepID) => async (dispatch, getState) => {
    let state = getState();
    if (state.testruns.savedSweepPlotsSweepID === sweepID) {
        return;
    }
    let result = await client.get(`sweepplot/saved/${sweepID}`);
    if (result) {
        dispatch({ type: TestController.SweepPlotsLoaded, payload: { sweepID, sweepPlots: result } });
    }
}

export const loadSavedPlot = (sweepID, plotID) => async (dispatch, getState) => {
    try {
        dispatch({ type: TestController.SweepPlotGenerated, payload: { plot: null } });
        let result = await client.get(`sweepplot/saved/${sweepID}/${plotID}`);
        let b = await result.blob();
        dispatch({ type: TestController.SweepPlotGenerated, payload: { plot: URL.createObjectURL(b) } });
    } catch (e) {
        dispatch({ type: TestController.Toast.Error, payload: e.message });
    }
}

export const deleteSavedPlot = (sweepID, plotID) => async (dispatch, getState) => {
    try {
        let s = getState()
        dispatch({ type: TestController.SweepPlotsLoaded, payload: { sweepID: sweepID, sweepPlots: s.testruns.savedSweepPlots.filter(p => p.id !== plotID) } });
        await client.del(`sweepplot/saved/${sweepID}/${plotID}`);
    } catch (e) {
        dispatch({ type: TestController.Toast.Error, payload: e.message });
    }
}


export const loadTestRunDetails = (id) => async (dispatch, getState) => {
    try {
        let state = getState();
        if (!state.testruns.testruns) {
            return;
        }
        let tr = state.testruns.testruns.find(r => r.id === id);
        if (tr && tr.detailsLoading === true) {
            return;
        }
        dispatch({ type: TestController.TestRunChanged, payload: Object.assign({ id: id, detailsLoading: true }) });
        let result = await client.get(`testruns/${id}/details`);
        if (result) {
            dispatch({ type: TestController.TestRunChanged, payload: Object.assign({}, result, { detailLoading: false, detailsAvailable: true }) });
        }
    } catch (e) {
        dispatch({ type: TestController.Toast.Error, payload: e.message });
    }
}

export const loadTestRunSweepMatrix = (sweepID) => async (dispatch, getState) => {
    try {
        let state = getState();
        let fetchMatrix = true;
        if (state.testruns.sweepMatrixSweepID === sweepID) {
            fetchMatrix = false;
        }
        if (fetchMatrix) {
            let result = await client.get(`testruns/sweepMatrix/${sweepID}`);
            dispatch({ type: TestController.TestRunSweepMatrixUpdated, payload: { matrix: result, sweepID: sweepID } });
        }
    } catch (e) {
        dispatch({ type: TestController.Toast.Error, payload: e.message });
    }
}


export const reloadTestResults = (id, trimZeroes, trimSamples, trimZeroesEnd) => async (dispatch) => {
    try {
        let result = await client.post(`testruns/${id}/results/recalc`, { trimZeroes: trimZeroes, trimSamples: trimSamples, trimZeroesEnd: trimZeroesEnd });
        if (result?.ok === true) {
            dispatch({ type: TestController.Toast.Success, payload: "Test run result recalculation started. Results will (re)appear when completed" });
            dispatch({ type: TestController.TestRunChanged, payload: { id: id, result: null } });
        }
    } catch (e) {
        dispatch({ type: TestController.Toast.Error, payload: e.message });
    }
}

export const terminateTestRun = id => async (dispatch) => {
    try {
        let result = await client.put(`testruns/${id}/terminate`);
        if (result?.ok === true) {
            dispatch({ type: TestController.Toast.Success, payload: `Test run ${id} termination requested` });
        }
    } catch (e) {
        dispatch({ type: TestController.Toast.Error, payload: e.message });
    }
}

export const terminateTestRunSweep = id => async (dispatch) => {
    try {
        let result = await client.get(`sweeps/${id}/cancel`);
        if (result?.ok === true) {
            dispatch({ type: TestController.Toast.Success, payload: `Test run sweep ${id} cancellation requested` });
        }
    } catch (e) {
        dispatch({ type: TestController.Toast.Error, payload: e.message });
    }
}


export const retrySpawning = id => async (dispatch) => {
    try {
        let result = await client.put(`testruns/${id}/retrySpawn`);
        if (result?.ok === true) {
            dispatch({ type: TestController.Toast.Success, payload: `Respawning of AWS agents requested for test run ${id}` });
        }
    } catch (e) {
        dispatch({ type: TestController.Toast.Error, payload: e.message });
    }
}

export const redownloadOutputs = id => async (dispatch) => {
    try {
        let result = await client.get(`testruns/${id}/redownloadOutputs`);
        if (result?.ok === true) {
            dispatch({ type: TestController.Toast.Success, payload: `Redownloading of S3 outputs requested for test run ${id}` });
        }
    } catch (e) {
        dispatch({ type: TestController.Toast.Error, payload: e.message });
    }
}


const mapListFields = (architectures, users) => tr => {

    let arch = architectures.find(a => a.id === (tr.architectureID || 'default'));
    let roleDesc = "";
    if (arch) {
        let roles = {};
        if (!tr.roleCounts) {
            for (var role of tr.roles) {
                if (roles[role.role]) {
                    roles[role.role] = roles[role.role] + 1;
                } else {
                    roles[role.role] = 1;
                }
            }
        } else {
            for (let rc of tr.roleCounts) {
                var role = arch.roles.find(r => r.role === rc.role);
                roles[rc.role] = rc.count;
            }
        }

        roleDesc = Object.keys(roles).map((k) => {
            var role = arch.roles.find(r => r.role === k);
            return `${roles[k]} ${role ? role.shortTitle : k.substr(0, 4)}`
        }).join(' / ');
    }

    let params = "";
    if (tr.fixedTxRate > 0 && tr.fixedTxRate < 1) {
        params += `Fixed: ${numeral(tr.fixedTxRate).format("#0%")} (${tr.loadGenInputCount}/${tr.loadGenOutputCount}) / `
    }
    if (tr.contentionRate > 0 && tr.contentionRate < 1) {
        params += `Contention: ${numeral(tr.contentionRate).format("#0[.][0][0][0][0]%")} / `
    }
    if (tr.preseedShards) {
        params += `Preseed: ${tr.preseedCount / 1000000}M / `
    }
    if (tr.invalidTxRate > 0) {
        params += `Invalid: ${tr.invalidTxRate} / `
    }
    if ((tr.architectureID.indexOf('default') > -1 && tr.shardReplicationFactor !== 2) ||
        (tr.architectureID.indexOf('2pc') > -1 && tr.shardReplicationFactor !== 3)) {
        params += `Shard Repl: ${tr.shardReplicationFactor} / `
    }

    if (tr.architectureID.indexOf('phase-two') > -1 && tr.loadGenTxType) {
        params += `TX Type: ${tr.loadGenTxType} / `
    }

    if (params.length > 0) {
        params = params.substr(0, params.length - 3);
    }
    let createdBy = users.find(u => u.thumbPrint === tr.createdByuserThumbprint);
    return {
        id: tr.id,
        params: params,
        sweepID: tr.sweepID,
        sweepOneAtATime: tr.sweepOneAtATime,
        createdByuserThumbprint: tr.createdByuserThumbprint,
        createdBy: createdBy?.name,
        createdByName: createdBy?.name,
        created: tr.created,
        notBefore: tr.notBefore,
        started: tr.started,
        completed: tr.completed,
        sortDate: `${tr.completed.startsWith('0001') ? (tr.notBefore.startsWith('0001') ? tr.created : tr.notBefore) : tr.completed}`,
        status: tr.status,
        architecture: arch,
        architectureName: arch?.name,
        avgThroughput: tr.avgThroughput,
        tailLatency: tr.tailLatency,
        details: tr.details,
        roles: roleDesc,
    };
}



const mapSweepFields = (architectures) => sweep => {

    let arch = architectures.find(a => a.id === (sweep.architectureID || 'default'));
    let roleDesc = "";
    let tr = sweep.firstRunData;
    if (arch) {
        let roles = {};
        if (!tr.roleCounts) {
            for (var role of tr.roles) {
                if (roles[role.role]) {
                    roles[role.role] = roles[role.role] + 1;
                } else {
                    roles[role.role] = 1;
                }
            }
        } else {
            for (let rc of tr.roleCounts) {
                var role = arch.roles.find(r => r.role === rc.role);
                roles[rc.role] = rc.count;
            }
        }

        roleDesc = Object.keys(roles).map((k) => {
            var role = arch.roles.find(r => r.role === k);
            return `${roles[k]} ${role ? role.shortTitle : k.substr(0, 4)}`
        }).join(' / ');
    }

    return Object.assign({}, sweep.firstRunData, sweep, { roles: roleDesc })
}

export const selectSweeps = createSelector(state => state.testruns.sweeps, state => state.architectures.architectures, (sweeps, architectures) => sweeps.map(mapSweepFields(architectures)).sort((a, b) => b.firstRun.valueOf() - a.firstRun.valueOf()));
export const selectActiveTestRunsList = createSelector(state => state.testruns.testruns, state => state.architectures.architectures, state => state.users.users, (testruns, architectures, users) => testruns.filter((tr) => ["Running"].indexOf(tr.status) > -1).map(mapListFields(architectures, users)).sort((a, b) => b.created.valueOf() - a.created.valueOf()));
export const selectQueuedTestRunsList = createSelector(state => state.testruns.testruns, state => state.architectures.architectures, state => state.users.users, (testruns, architectures, users) => testruns.filter((tr) => ["Queued"].indexOf(tr.status) > -1).map(mapListFields(architectures, users)).sort((a, b) => b.created.valueOf() - a.created.valueOf()));
export const selectCompletedTestRunsList = createSelector(state => state.testruns.testruns, state => state.architectures.architectures, state => state.users.users, (testruns, architectures, users) => testruns.filter((tr) => tr.status === "Completed").map(mapListFields(architectures, users)).sort((a, b) => b.completed.valueOf() - a.completed.valueOf()));
export const selectFailedTestRunsList = createSelector(state => state.testruns.testruns, state => state.architectures.architectures, state => state.users.users, (testruns, architectures, users) => testruns.filter((tr) => ["Canceled", "Aborted", "Failed", "Interrupted"].indexOf(tr.status) > -1).map(mapListFields(architectures, users)).sort((a, b) => b.completed.valueOf() - a.completed.valueOf()));
export const selectQueuedTestRunCount = createSelector(state => state.testruns.testruns, testruns => testruns.filter(tr => tr.state === 'Queued').length);
export const selectTestRunLast24hCount = createSelector(state => state.testruns.testruns, testruns => testruns.filter(tr => new Date(tr.completed).valueOf() >= new Date().valueOf() - 86400000).length);
export const selectNewestTestRun = createSelector(state => state.testruns.testruns, (testruns) => testruns.filter((tr) => tr.status === "Completed").sort((a, b) => new Date(b.completed).valueOf() - new Date(a.completed).valueOf())[0] || {});
export const selectScheduledRunRoles = createSelector(state => state.testruns.scheduledTestRun, state => state.architectures.architectures, state => state.agents.launchTemplates, (tr, architectures, launchTemplates) => {
    const architecture = architectures.find(a => a.id === (tr.architectureID || 'default'))
    return tr.roles.sort((a, b) => {
        return (
            a.role.localeCompare(b.role) ||
            a.roleIdx - b.roleIdx
        );
    })
        .map((r) => {
            r = Object.assign({}, r, { launchTemplate: launchTemplates.find(lt => lt.id === r.awsLaunchTemplateID) })
            var role = architecture.roles.find(
                (ar) => ar.role === r.role
            );
            var roleTitle = role ? role.title : "";
            return {
                Role: roleTitle,
                Index: r.roleIdx,
                Agent: Object.assign({}, r),
                Fail: Object.assign({}, r),
                Delete: Object.assign({}, r),
            };
        });
})

export const selectScheduledRunSweepRoles = createSelector(state => state.testruns.scheduledTestRun, state => state.architectures.architectures, (tr, architectures) => {
    const architecture = architectures.find(a => a.id === (tr.architectureID || 'default'))
    return tr.sweepRoles.sort((a, b) => {
        return (
            a.role.localeCompare(b.role) ||
            a.roleIdx - b.roleIdx
        );
    })
        .map((r) => {
            var role = architecture.roles.find(
                (ar) => ar.role === r.role
            );
            var roleTitle = role ? role.title : "";
            return {
                Role: roleTitle,
                Index: r.roleIdx,
                Agent: Object.assign({}, r),
                Delete: Object.assign({}, r),
            };
        });
})
import {CSelect} from "@coreui/react";

const Selector = (props) => {
  return (
    <CSelect onChange={props.onChange} value={props.value}>
      <option value="">-- SELECT --</option>
      {props.values.map((v) => {
        return (
          <option key={props.valueFunc(v)} value={props.valueFunc(v)}>
            {props.displayFunc(v)}
          </option>
        );
      })}
    </CSelect>
  );
};

export default Selector;


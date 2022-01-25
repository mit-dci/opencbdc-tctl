import { useSelector } from "react-redux";

const User = (props) => {
    const usr = useSelector(state => {
        return state.users.users.find((u) => u.thumbPrint === props.thumbPrint)
    });
    if(usr){
        return <span>{usr.name} ({usr.org})</span>
    } else {
        return <span>Unknown</span>
    }

}

export default User;
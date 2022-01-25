import * as React from 'react'

class AutoScrollingTextarea extends React.Component {
    componentDidMount() {
        if (this.element && this.props.autoScroll !== false) {
            this.element.scrollTop = this.element.scrollHeight
        }
    }

    componentDidUpdate(prevProps, prevState) {
        if (prevProps.value !== this.props.value && this.element && this.props.autoScroll !== false) {
            this.element.scrollTop = this.element.scrollHeight
        }
    }

    render() {
        return <textarea {...this.props} readOnly={true} ref={e => this.element = e} />
    }
}
export default AutoScrollingTextarea
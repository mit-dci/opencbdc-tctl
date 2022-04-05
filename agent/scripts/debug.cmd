handle SIGHUP nostop pass
handle SIGQUIT nostop pass
handle SIGPIPE nostop pass
handle SIGALRM nostop pass
handle SIGTERM nostop pass
handle SIGUSR1 nostop pass
handle SIGUSR2 nostop pass
handle SIGCHLD nostop pass
set print thread-events off
run
thread apply all bt
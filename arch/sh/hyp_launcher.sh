#!/bin/bash

SCREEN_NAME="hyperionweb"
DIR_ROOT="/srv/hyperionweb"
SERVICE="hyperionweb.go"
USER="hyperionweb"

function start {
	if [ `whoami` = root ]
  	then
    	su - $USER -c "cd $DIR_ROOT ; screen -h 5000 -AdmS $SCREEN_NAME bash -c 'go run $SERVICE $DIR_ROOT'"
  	else
    	cd $DIR_ROOT
    	screen -AdmS $SCREEN_NAME bash -c "go run $SERVICE $DIR_ROOT"
  	fi
}

function stop {
  	if ! status; then echo "$SCREEN_NAME could not be found. Probably not running."; exit 1; fi

  	if [ `whoami` = root ]
  	then
    	tmp=$(su - $USER -c "screen -ls" | awk -F . "/\.$SCREEN_NAME\t/ {print $1}" | awk '{print $1}')
    	su - $USER -c "screen -r $tmp -X quit"
	else
    	screen -r $(screen -ls | awk -F . "/\.$SCREEN_NAME\t/ {print $1}" | awk '{print $1}') -X quit
  	fi
}

function status {
  	if [ `whoami` = root ]
  	then
    	su - $USER -c "screen -ls" | grep [.]$SCREEN_NAME[[:space:]] > /dev/null
  	else
    	screen -ls | grep [.]$SCREEN_NAME[[:space:]] > /dev/null
  	fi
}

function console {
  	if ! status; then echo "$SCREEN_NAME could not be found. Probably not running."; exit 1; fi

  	if [ `whoami` = root ]
  	then
    	echo "Running as root"
    	tmp=$(su - $USER -c "screen -ls" | awk -F . "/\.$SCREEN_NAME\t/ {print $1}" | awk '{print $1}')
    	su - $USER -c "screen -dm -r $tmp"
  	else
    	echo "Running as someone else"
    	echo $USER
    	screen -dm -r $(screen -ls | awk -F . "/\.$SCREEN_NAME\t/ {print $1}" | awk '{print $1}')
  	fi
}

function usage {
  	echo "Usage: service the hyperion launcher {start|stop|status|restart|console}"
  	echo "On console, press CTRL+A then D to stop the screen without stopping the server."
}

case "$1" in

	start)
	    echo "Starting $SCREEN_NAME..."
	    start
	    sleep 0.5
	    echo "$SCREEN_NAME started successfully"
  	;;

  	stop)
    	echo "Stopping $SCREEN_NAME..."
    	stop
   		sleep 0.5
    	echo "$SCREEN_NAME stopped successfully"
  	;;
 
  	restart)
	    echo "Restarting $SCREEN_NAME..."
	    status && stop
	    sleep 0.5
	    start
	    sleep 0.5
	    echo "$SCREEN_NAME restarted successfully"
  	;;

  	status)
	    if status
	    then echo "$SCREEN_NAME is UP"
	    else echo "$SCREEN_NAME is DOWN"
	    fi
  	;;
 
  	console)
	    echo "Open console on $SCREEN_NAME..."
	    console
  	;;

  	*)
    	usage
    	exit 1
  	;;

esac

exit 0


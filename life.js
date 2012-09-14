// vim:sw=3:ts=3:fdm=indent

var image = document.getElementById("life")
var delay, realDelay
var stop = 0
var stopLog = document.getElementById("stopLog")
var startStopLog = document.getElementById("startStopLog")
var delayDiv = document.getElementById("delay")
delay = delayDiv.innerHTML
realDelay = delay

function getUpdates() {
	var xmlhttp;
	if (window.XMLHttpRequest) {
		// code for IE7+, Firefox, Chrome, Opera, Safari
		xmlhttp=new XMLHttpRequest();
   } else {
		// code for IE6, IE5
	   xmlhttp=new ActiveXObject("Microsoft.XMLHTTP");
	}

	xmlhttp.onreadystatechange=function() {
		if (xmlhttp.readyState==4 && xmlhttp.status==200) {
			eval(xmlhttp.responseText);

			// Send another long poll
			xmlhttp.open("GET", "updates", true);
			xmlhttp.send();
		}
	}

	xmlhttp.open("GET", "updates", true);
	xmlhttp.send();
}

function sendButton(event) {
	var xmlhttp;
	if (window.XMLHttpRequest) {
		// code for IE7+, Firefox, Chrome, Opera, Safari
		xmlhttp=new XMLHttpRequest();
   } else {
		// code for IE6, IE5
	   xmlhttp=new ActiveXObject("Microsoft.XMLHTTP");
	}
	// Response via getUpdates() loop
	xmlhttp.open("GET", "button?title=" + event.target.id, true);
	xmlhttp.send();
}

function nextImage() {
	image.src="/life.png"
	if (!stop) {
		stopLog.innerHTML = stop
		setTimeout(function(){nextImage()}, delay)
	}
}

function refresh(o) {
	delay = o.delay
	delayDiv.innerHTML = Math.floor(delay)
	if (stop != o.stop) {
		if (o.stop) {
			stopLog.innerHTML = stop
			startStopLog.innerHTML = "stopping"
			stop = o.stop
		} else {
			stopLog.innerHTML = stop
			startStopLog.innerHTML = "running"
			stop = o.stop
			// This is a race condition: what if this fires before the rest of
			// the function finishes?
			setTimeout(function(){nextImage()}, 0)
		}
	}
}

setTimeout(function(){nextImage()}, delay)

// Start the "update loop"
setTimeout(function(){getUpdates()}, 0)

// vim:sw=3:ts=3:fdm=indent

var image = document.getElementById("life")
var delay
var startStopLog = document.getElementById("startStopLog")
var delayDiv = document.getElementById("delay")
var log = document.getElementById("Log")
var log2 = document.getElementById("Log2")
delay = delayDiv.innerHTML

var nextImageIntervalID

function getUpdates() {
	var xmlhttp
	if (window.XMLHttpRequest) {
		// code for IE7+, Firefox, Chrome, Opera, Safari
		xmlhttp=new XMLHttpRequest()
   } else {
		// code for IE6, IE5
	   xmlhttp=new ActiveXObject("Microsoft.XMLHTTP")
	}

	xmlhttp.onreadystatechange=function() {
		alert(xmlhttp.readyState +" "+xmlhttp.status)
		if (xmlhttp.readyState == 4) {
			if (xmlhttp.status == 200) {
				// alert("response is " + xmlhttp.responseText)
				eval(xmlhttp.responseText)
			} else {
				// Only poll every second if we're getting errors
				// setTimeout(function(){getUpdates()}, 1000)
			}
			// Send another long poll
			xmlhttp.open("GET", "updates", true)
			xmlhttp.send()
		}
	}

	xmlhttp.open("GET", "updates", true)
	xmlhttp.send()
}

function sendButton(event) {
	var xmlhttp
	if (window.XMLHttpRequest) {
		// code for IE7+, Firefox, Chrome, Opera, Safari
		xmlhttp=new XMLHttpRequest()
   } else {
		// code for IE6, IE5
	   xmlhttp=new ActiveXObject("Microsoft.XMLHTTP")
	}
	// Response via getUpdates() loop
	xmlhttp.open("GET", "button?title=" + event.target.id, true)
	xmlhttp.send()
}

function nextImage() {
	image.src="/life.png"
}

function refresh(o) {
	log.innerHTML = o.delay + " " + o.stop
	delay = Math.floor(o.delay)
	delayDiv.innerHTML = delay
	if (o.stop) {
		startStopLog.innerHTML = "stopping"
		clearInterval(nextImageIntervalID)
	} else {
		startStopLog.innerHTML = "running"
		clearInterval(nextImageIntervalID)
		nextImage()
		nextImageIntervalID = setInterval(function(){nextImage()}, delay)
	}
}

// Start the "update loop"
setTimeout(function(){getUpdates()}, 1)
refresh({"delay":delay, "stop":0})

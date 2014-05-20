var sampleController = {
	init: function(){
		susi.events.subscribe("*",this.logCallback);
	},
	logCallback: function(evt){
		susi.log("Logging: "+JSON.stringify(evt.Topic)+" : "+JSON.stringify(evt.Payload));
	}
}

sampleController.init();

susi.events.publish("foo",{this: "is it"});


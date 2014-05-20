var sampleController = {
	init: function(){
		susi.events.subscribe("foo",this.fooCallback);
	},
	fooCallback: function(evt){
		susi.log("Foo Callback: "+JSON.stringify(evt.Payload));
	}
}

sampleController.init();

susi.events.publish("foo",{this: "is it"});


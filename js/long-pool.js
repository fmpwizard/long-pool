(function(window, undefined) {
	"use strict";

	window.init = (function() {
		var self = this;
    	// "private" vars
	    var ajaxSuccess = function(resp) {
	  		console.log("resp was " , resp);
		};
	    var ajaxError = function(resp) {
	  		console.log("Error: resp was " , resp);
		};
	    console.log("cometId " + cometId);
	    console.log("index " + index);
	    
	    var makeAjax = function(_index, _cometId){
	    	$.ajax({
		  		url: "/comet?index="+_index+"&cometid="+_cometId,
		  		context: document.body,
		  		timeout: 2000,
		  		dataType: 'json',
		  		success: ajaxSuccess,
		  		error: ajaxError
			}).done(function() {
		  		console.log("Done");
				//makeAjax(resp.LastIndex,cometId);
			});
		};
		makeAjax(index,cometId);
		makeAjax(1,cometId);

	})();
})(this);

//http://127.0.0.1:7070/index
//http://127.0.0.1:7070/add?data=Diego+was+here+1&cometid=0.73059868633358404644

//http://127.0.0.1:7070/comet?index=3&cometid=0.89506036657873655482

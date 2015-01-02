(function(window, undefined) {
	"use strict";

	window.init = (function() {
		var self = this;
    	// "private" vars
	    var doCallback = function(resp) {
	    	if (resp.Res){
		    	$.each(resp.Res, function(_, item){
		  			if (item.Value){
		  				console.log(item.Value);	
		  			}
		  		});	
	    	}
	  		
		};
	    var ajaxSuccess = function(resp) {
	  		doCallback(resp);
		};
	    var ajaxError = function() {
	  		makeAjax(0, cometId);
		};
	    console.log("cometId " + cometId);	    
	    var makeAjax = function(_index, _cometId){
	    	$.ajax({
		  		url: "/comet?index="+_index+"&cometid="+_cometId,
		  		context: document.body,
		  		timeout: 6000,
		  		dataType: 'json',
		  		success: ajaxSuccess,
		  		error: ajaxError
			}).done(function(resp) {
				makeAjax(resp.LastIndex,cometId);
			});
		};
		makeAjax(index,cometId);
	})();
})(this);

//http://127.0.0.1:7070/index
//http://127.0.0.1:7070/add?data=Diego+was+here+1&cometid=0.73059868633358404644

//http://127.0.0.1:7070/comet?index=3&cometid=0.89506036657873655482

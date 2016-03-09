function postForm(path, serialized) {
	var xmlhttp = new XMLHttpRequest();
	xmlhttp.open("POST", path, false);
	xmlhttp.setRequestHeader("Content-type","application/x-www-form-urlencoded");
	xmlhttp.send(serialized);
	return xmlhttp.responseText;
}

// override POST requests
$(document).on('submit', '#color_form', function(e) {
	e.preventDefault();
	handleResult(postForm("/set_color_name", $(this).serialize()));
});

$(document).on('submit', '#static_form', function(e) {
	e.preventDefault();
	handleResult(postForm("/set_static", $(this).serialize()));
});

$(document).on('submit', '#effect_form', function(e) {
	e.preventDefault();
	handleResult(postForm("/set_effect", $(this).serialize()));
});

$(document).on('submit', '#valueGain_form', function(e) {
	e.preventDefault();
	handleResult(postForm("/set_value_gain", $(this).serialize()));
});

$(document).on('submit', '#clear_form', function(e) {
	e.preventDefault();
	handleResult(postForm("/do_clear", $(this).serialize()));
});

$(document).on('submit', '#restart_form', function(e) {
	e.preventDefault();
	postForm("/do_restart", $(this).serialize());
});

$(document).on('submit', '#start_form', function(e) {
	e.preventDefault();
	postForm("/do_start", $(this).serialize());

	location.reload();
});

$(document).on('submit', '#stop_form', function(e) {
	e.preventDefault();
	postForm("/do_stop", $(this).serialize());

	location.reload();
});

$(document).on('submit', '#restart_form', function(e) {
	e.preventDefault();
	postForm("/do_restart", $(this).serialize());

	location.reload();
});


function handleResult(result) {

	if (result == "" || !JSON.parse(result).success) {
		$('.error').fadeIn(400).delay(2000).fadeOut(400);
	} else {
		$('.result').fadeIn(400).delay(2000).fadeOut(400); //fade out after 2 seconds
	}
}

$(function() { $('.colorpicker').wheelColorPicker(); }); // colorpicker init

// list of colors
var color_array = [];
$.get('http://andreashenriksson.se:1234/js/colors.txt', function(data) {
	var hasNumber = /\d/;
    var lines=data.split('\n');
    for (var i = 0; i < lines.length; i++){
        var item=lines[i];
        //item = item.replace(/[0-9]/g, ''); // removes names with numbers but creates duplicates

        item = item.replace(/(\w+\s+\w+\s+\w+?)$/, ""); // remove rgb values
        item = item.trimRight();
        if (hasNumber.test(item) || (item[0] === item[0].toUpperCase())) { // ignore all lines containing numbers or uppcases
        	continue;
        }

    	color_array.push([item]);
    	//console.log(item); // for development
    }
});
/**
 *
 * You can write your JS code here, DO NOT touch the default style file
 * because it will make it harder for you to update.
 *
 */

"use strict";
const CoordDefault = 0.0;
const DelayDefault = 1000;
const DurationDefault = 3000;
const KeepAliveDefault = 30;
const ZeroDefault = 0;
const ACKTimeoutDefault = 2;
const UplinkIntervalDefault = 10;
const rangeDefault = 10000;
const MaxValueCounter = 16384;
const UnConfirmedData_uplink ="UnConfirmedDataUp"
const ConfirmedData_uplink ="ConfirmedDataUp"

var LengthEUI64 = 16;
var LengthDevAddr = 8;

var StateSimulator = false;//true in running

//checks
var DataRates = [];
var minFrequency = 100;
var maxFrequency = 100;
var TablePayload = [];
var TablePayloadDT = [];
var frequencyRX2Default = 0;
var dataRateRX2Default = 0;

//maps
var MapGateway;
var MapDevice;
var MapHome;
var MapModal;

var MarkerGateway = {};
var MarkerDevice = {};
var MarkersHome = new Map();
var MarkerModal = {};
var Circle;

var TurnMap = 0;

var Gateways = new Map();
var Devices = new Map();

//socket
var socket = io({
                path:'/socket.io/',
                reconnectionDelay:5000});

var url = window.origin;

$(document).ready(function(){

    Initmap();
    setTimeout(() => {
        MapModal.invalidateSize();
    }, 500);

    MapGateway.on('click',function(e){

        if ($(this._container).parents("#location").find("#coords").find("input").prop("disabled"))
            return;

        Click_Change_Marker(e);
    });
    
    MapDevice.on('click',function(e){

        if ($(this._container).siblings("#coords").find("input").prop("disabled"))
            return;

        Click_Change_Marker(e);
    });

    MapModal.on('click',Click_Change_Marker);

    $(window).resize(function(){
        setTimeout(() => {
            MapModal.invalidateSize();
        }, 500);
    });

    // ********************** socket event *********************

    socket.on('connect',()=>{
        Init();
    });

    socket.on('disconnect',()=>{
        
        StateSimulator = false;
        $("#state").attr("src", "img/red_circle.svg")
        $(".btn-play").parent("button").removeClass("hide");
        $(".btn-stop").parent("button").addClass("hide");
        
    });

    socket.on('console-sim',(data)=>{

        var row = "<p class=\"text-break text-start bg-secondary m-0\">"+data.message+"</p>";
        $('#console-body').append(row);

        $('#console-body').animate({
            scrollTop: $('#console-body').get(0).scrollHeight
        }, 0);

    });

    socket.on('console-error',(data)=>{
      
        var row = "<p class=\"text-break text-white bg-danger m-0\">"+data.message+"</p>";             
        $('#console-body').append(row);

        $('#console-body').animate({
            scrollTop: $('#console-body').get(0).scrollHeight
        }, 0);

    });

    socket.on('log-dev',(data)=>{

        var classesP = $("[name=\""+data.name+"\"]").attr("class");//p
        
        $("span[data-name=\""+data.name+"\"]").attr("class");

        var row;

        if (classesP == undefined)
            row = "<p class=\"text-break text-start text-info clickable me-1 mb-0\" name=\""+data.name+"\" data-name=\""+data.name+"\">"+data.message+"</p>";
        else
            row = "<p class=\""+classesP+"\" name=\""+data.name+"\" data-name=\""+data.name+"\">"+data.message+"</p>";
    
        $('#console-body').append(row);

        $('#console-body').animate({
            scrollTop: $('#console-body').get(0).scrollHeight
        }, 0);

    });

    socket.on('log-gw',(data)=>{ 

        var valueName = "gw-"+data.name;     
        var classesP = $("[name="+valueName+"]").attr("class");//p
        var row;

        if (classesP == undefined)
            row = "<p class=\"text-break clickable text-start text-warning me-1 mb-0\" data-name=\""+valueName+"\" name=\""+valueName+"\">"+data.message+"</p>";
        else
            row = "<p class=\""+classesP+"\" name="+valueName+" data-name=\""+valueName+"\">"+data.message+"</p>";
    
        $('#console-body').append(row);

        $('#console-body').animate({
            scrollTop: $('#console-body').get(0).scrollHeight
        }, 0);

    });

    socket.on('save-status',(data)=>{

        var dev = Devices.get(data.devEUI);

        dev.info.devAddr = data.devAddr;
        dev.info.nwkSKey = data.nwkSKey;
        dev.info.appSKey = data.appSKey;
        dev.info.status.fcntDown = data.fcntDown;
        dev.info.status.infoUplink.fcnt = data.fcnt;

    });

    socket.on('response-command',(data)=>{
        Show_iziToast(data,"");
    });

    // ********************** nav bar *********************

    $(".btn-play").parent("button").on("click",function(){
        
        if (!socket.connected){
            Show_ErrorSweetToast("Socket not connected","");
            return;
        }

        if (StateSimulator){
            Show_ErrorSweetToast("Simulator already run","");
            return;
        }

        $("#state").attr("src","img/yellow_circle.svg");

        $.ajax({
            url:url+"/api/start",
            type:"GET",
            headers:{
                "Access-Control-Allow-Origin":"*"
            }
        }).done((data)=>{
            
            if (data){
                
                StateSimulator = true;

                Show_iziToast("Simulator started","");
                $("#state").attr("src","img/green_circle.svg");
                $(".btn-play").parent("button").toggleClass("hide");
                $(".btn-stop").parent("button").toggleClass("hide");

            }               
            else{

                Show_ErrorSweetToast("Error","Simulator didn't started");
                $("#state").attr("src","img/red_circle.svg");

            }     
                  
        }).fail((data)=>{

            $("#state").attr("src","img/red_circle.svg");
            Show_ErrorSweetToast("Unable to start the simulator",data.statusText); 
            
        });

    });

    $(".btn-stop").parent("button").on("click",function(){

        if(!StateSimulator){
            Show_ErrorSweetToast("Simulator already stop","");
            return;
        }
        
        StateSimulator = false;
        
        $("#state").attr("src","img/yellow_circle.svg");

        $.get(url+"/api/stop",{
        
        }).done((data)=>{
        
            if (data){

                $("#state").attr("src","img/red_circle.svg");
                $(".btn-play").parent("button").toggleClass("hide");
                $(".btn-stop").parent("button").toggleClass("hide");

                Show_iziToast("Simulator stopped","");
            }
            else{
                $("#state").attr("src","img/green_circle.svg");
            }   

        }).fail((data)=>{

            $("#state").attr("src","img/green_circle.svg");
            Show_ErrorSweetToast("Unable to stop the simulator",data.statusText); 

        });

    });
    
    // ********************** sidebar *********************
    
    $("#sidebar a").on("click", function () {
        
        if($(this).hasClass("has-dropdown"))
            return;

        $(".main-content > div").removeClass("show active");
        $($(this).data("tab")).addClass("show active");

        if (this.id == "home-tab"){
            TurnMap = 0;
            $(".section-header h1").text("LWN Simulator");

            LoadListHome();
        }

        $(this).parent("li").siblings("li").removeClass("active");
        $(this).parent("li").addClass("active");
                            
    });

    $("#sidebar a:not(.has-dropdown)").on('click',function(){
        
        var parentUL = $(this).parents("ul.dropdown-menu");
        if (parentUL.length != 0)
            return;

        $(".main-sidebar .sidebar-menu li.dropdown > .dropdown-menu").slideUp(500, function() {            
            
            let a = setInterval(function() {

                var sidebar_nicescroll = $(".main-sidebar").getNiceScroll();
                if(sidebar_nicescroll != null)
                    sidebar_nicescroll.resize();

                }, 10);
            
                setTimeout(function() {
                    clearInterval(a);
                }, 600);
        });

    });

    $("[data-tab*=dev]").on("click",function(){

        TurnMap = 1;

        CleanInputDevice();

        MapDevice.invalidateSize();

        $(".section-header h1").text($(this).text())
        $("#header-sidebar-dev h4").text("Device's Settings");
    });

    $("[data-tab*=gw]").on("click",function(){

        TurnMap = 2;

        CleanInputGateway();

        $(".section-header h1").text($(this).text())
    });

    // ********************** home *********************
    
    $("#container-header-accordion").on("click", function(){
        $("#console-body").toggleClass("show");
    });

    $(".btn-clean").on("click",function(){
        $("#console-body").empty();
        $(this).blur();
    })

    $("#console-body").on('click',"p",function(){
    
        var val = $(this).data("name");
        if (val.includes("gw")){
            $("[name=\""+val+"\"]").toggleClass("text-warning bg-warning");
        }else{
            $("[name=\""+val+"\"]").toggleClass("text-info bg-info");
        }
        
    });

    //click item list
    $("#list-home").on("click","a",function(){ 

        var address = $(this).attr("data-addr");
        var marker = MarkersHome.get(address).Marker;
        var dev = Devices.get(address);
        
        ChangeView(marker.getLatLng(),11)
        marker.openPopup();

        if(dev != undefined)
            DrawRange(marker.getLatLng(), dev.info.configuration.range);
    });

    // ********************** map *********************

    //Latitude
    $("[name^=input-latitude]").on('keyup',function(){

        $(this).val($(this).val().replaceAll(',','.'));

        var val_lng = $(this).parents("#coords").find(" [name=input-longitude]").val();

        var latlng = [Number($(this).val()), Number(val_lng)];

        if ($(this).val() == "") 
            $(this).val(0)

        var valid = IsValidNumber($(this).val(),-90.01,90.01);
        
        ValidationInput($(this), valid);

        ChangeView(latlng,10);
        ChangePositionMarker(-1,latlng);
    });

    //Longitude
    $("[name^=input-longitude]").on('keyup',function(){
        $(this).val($(this).val().replaceAll(',','.'));

        var val_lng = $(this).parents("#coords").find("[name=input-latitude]").val();

        var latlng =[Number($(this).val()), Number(val_lng)];

        if ($(this).val() == "") 
            $(this).val(0)

        var valid =  IsValidNumber($(this).val(),-180.01,180.01);
        
        ValidationInput($(this), valid);

        ChangeView(latlng,10);
        ChangePositionMarker(-1,latlng);
    });

    // ********************** sidebar/dropdown: list devices *********************
    
    //click item list
    $("#list-devices").on("click","tr .clickable",function(){ 

        var address = $(this).parents("tr").attr("data-addr");

        CleanInputDevice();
        LoadDevice(Devices.get(address));

        $("#header-sidebar-dev h4").text($(this).text());
    
    });
    
    // ********************** sidebar/dropdown: add new device *********************

    $("#location-tab").on("click",function(){
        setTimeout(()=>{
            MapDevice.invalidateSize();
        },300);
        
    });

    $("[name=input-devEUI]").on('blur keyup',function(){

        if ($(this).val().length == 0){
            $(this).removeClass("is-valid is-invalid"); 
            return 
        }

        var valid = IsValidAddress($(this).val(), true)  

        ValidationInput($(this),valid); 
      
    });

    //generate devEUI
    $('[name=btn-new-devEUI]').on('click',function(){
        Click_GenerateAddress($("[name=input-devEUI]"), LengthEUI64);
    });

    $("#region").on('change',function(){
        $(this).removeClass("is-valid is-invalid"); 

        if(Number($(this).val()) == -1)
            return;

        SetParameters(Number($(this).val()), false, null);
        
    });

    //otaa flag
    $('#checkbox-otaa-dev').on('click',function(){

        ChangeStateActivation($(this).prop('checked'))

        $("[name=input-devAddr]").val("");
        $("[name^=input-key]").val("");
        
        $("[name^=input-key]").removeClass("is-valid is-invalid");
        $("[name=input-devAddr]").removeClass("is-valid is-invalid");

        $(".btn-watch > img").removeClass("seeOFF");
        $("[name^=input-key]").attr("type","password");

    });

    //dev addr
    $('[name=input-devAddr]').on('blur keyup',function(){

        if ($(this).val().length == 0){
            $(this).removeClass("is-valid is-invalid"); 
            return 
        }

        var valid = IsValidAddress($(this).val(), false)  

        ValidationInput($(this),valid); 

    });

    //generate devAddr
    $('[name=btn-new-devAddr]').on('click',function(){
        Click_GenerateAddress($("[name=input-devAddr]"),LengthDevAddr);
    });

    //watch key
    $('[name=btn-watch]').on('click',function(){
        SeeKey($(this).siblings("[name^=input-key-]"), $(this).children("img"));      
    });

    //check key value
    $("[name^=input-key-]").on("keyup", function(e){
        Check_key(this,e);
    });

    //generate key
    $('[name=btn-new-key]').on('click',function(){

        var key = GenerateKey();
        $(this).siblings("[name^=input-key-]").val(key);

        ValidationInput($(this).siblings("[name^=input-key-]"),true);

    });

    //delay, duration (rx1/rx2), freq rx2 check value
    $("[name ^=input-rx-]").on("blur keyup", function(){

        var valid = IsValidNumber($(this).val(),0, Infinity);
        ValidationInput($(this),valid);

    });

    //controllo sula frequenza
    $("[name=input-frequency-rx-2]").on("blur keyup", function(){

        var valid = IsValidNumber($(this).val(),minFrequency, maxFrequency);
        ValidationInput($(this), valid);
        
    });

    //ack timeout check value
    $("[name=input-ackTimeout]").on("blur keyup", function(){

        var valid = IsValidNumber($(this).val(),0, 4);;
        ValidationInput($(this),valid);

    });

    //Class B , C
    $("[name=input-flag]").on('click',function(){

        var flag = $(this).data("class");
        var checked = $(this).prop("checked");

        if (flag == "B") {

            if (!checked)
                $("#classC [name=input-flag]").prop({"disabled":false,"readonly":false});
            else
                $("#classC [name=input-flag]").prop({"checked":false,"disabled":true,"readonly":true});
        
        }else{

            if (!checked)
                $("#classB [name=input-flag]").prop({"disabled":false,"readonly":false});
            else
                $("#classB [name=input-flag]").prop({"checked":false,"disabled":true,"readonly":true});
        
        }
    });

    //Fport check value
    $("[name=input-fport]").on("blur keyup", function(e){
        var valid = IsValidNumber($(this).val(), 0, 224);
        ValidationInput($(this),valid);
    });

    //retransmission check value
    $("[name=input-retransmission]").on("blur keyup", function(e){
        var valid = IsValidNumber($(this).val(), -1, Infinity);
        ValidationInput($(this),valid);
    });

     //flag validation
    $("[name=input-validate-counter]").on("click",function(){

        var value = $(this).prop("checked");
        $("[name=input-fcnt-downlink]").val("");
        $("[name=input-fcnt-downlink]").removeClass("is-valid is-invalid")
        $("[name=input-fcnt-downlink]").prop({"disabled":value, "readonly":value});
           
    });

    //Counters
    $("[name^=input-fcnt]").on("blur keyup",function(){
        var valid = IsValidNumber($(this).val(), -1, 163845);
        ValidationInput($(this),valid);
    });

    $("[name=input-range]").on("blur keyup",function(){
        var valid = IsValidNumber($(this).val(), 0, Infinity);
        ValidationInput($(this),valid);
    });

    $("[name=input-sendInterval]").on("blur keyup", function(e){
        var valid = IsValidNumber($(this).val(),0, Infinity);
        ValidationInput($(this),valid);
    });

    //buttons
    $("[name=btn-delete-dev").on('click',function(){
        Click_DeleteDevice();
    });

    $("[name=btn-edit-dev").on('click',function(){
        Click_Edit(this, false);
    });

    $("[name=btn-save-dev]").on('click',function(){
        Click_SaveDevice();
    });

    // ********************** sidebar/dropdown: list gateways *********************
    //click item list
    $("#list-gateways").on("click","tr .clickable",function(){
        
        var address = $(this).parents("tr").attr("data-addr");

        CleanInputGateway();
        LoadGateway(Gateways.get(address));
    });

    // ********************** sidebar/dropdown: add new gateway *********************

    //input MAC 
    $("[name=input-MAC-gw]").on("blur keyup",function(e){

        if ($(this).val().length == 0){
            $(this).removeClass("is-valid is-invalid"); 
            return 
        }

        var valid = IsValidAddress($(this).val(), true)  

        ValidationInput($(this),valid); 

    });

    //generate new mac address
    $("[name=btn-new-MACAddress]").on("click",function(){
        Click_GenerateAddress($(this).siblings("[name=input-MAC-gw]"),LengthEUI64);
    });

    $("#choose-type button").on('click',function(){

        if ($(this).prop("disabled"))
            return;

        setTimeout(()=>{
            MapGateway.invalidateSize();
        },300);


        $(this).siblings().removeClass("active");
        $(this).addClass("active");

        $("#info-gw").removeClass("hide");
        
        if ($(this).attr("id") == "virtual-gw"){
            $("#info-virtual-gw").removeClass("hide");
            $("#info-real-gw").addClass("hide");
        }            
        else{
            $("#info-real-gw").removeClass("hide");
            $("#info-virtual-gw").addClass("hide");
        }
            
    });

   //keep Alive
    $("[name=input-KeepAlive]").on("blur keyup",function(){

        var value = $(this).val();
        var valid = IsValidNumber(value,0, Infinity);     
        
        ValidationInput($(this),valid);

    }); 

     //btn save
   $("[name=btn-save-gw]").on("click",function(){
        Click_SaveGateway($(this)); 
    });

    //btn edit
    $("[name=btn-edit-gw]").on("click",function(){
        Click_Edit(this, true);
    });

    //btn delete
    $("[name=btn-delete-gw]").on("click",function(){
        Click_DeleteGateway()
    });

    // ********************** sidebar: gateway bridge *********************
    $('#bridge-tab').on('click', function(){

        $.ajax({
            url: url+"/api/bridge/",
            type:"GET",
            headers:{
                "Access-Control-Allow-Origin":"*"
            }

        }).done((data)=>{

            if (data.ip != "")
                $('[name=input-IP-bridge]').val(data.ip)
                         
            if (data.port != "")
                $('[name=input-port-bridge]').val(data.port)                
            
        }).fail((data)=>{
            Show_ErrorSweetToast("Unable to load info of gateway bridge", data.statusText);       
        });

    });

    //btn save bridge's info
    $("[name=save-bridge]").on("click",function(){

        if (!CanExecute()) {
            Show_ErrorSweetToast("Simulator in running", "Unable change data");
            return
        }
        
        var address = $("[name=input-IP-bridge]").val();
        var port = $("[name=input-port-bridge]").val();

        //validation
        var validAddr = IsValidURL(address) || IsValidIP(address);  
        var validPort = port < 65536 && port > 0 ? true : false;

        var val = validAddr && validPort;
        
        if(!val){

            ValidationInput($("[name=input-IP-bridge]"), validAddr)
            ValidationInput($("[name=input-port-bridge]"), validPort)
            
            Show_ErrorSweetToast("Error", "Values are incorrect")

            return
        }

        //create file JSON
        var jsonData = JSON.stringify({
            "ip" : address,
            "port" : port
        });

        //ajax
        $.post(url + "/api/bridge/save", jsonData,"json")
        .done((data)=>{

            var header = IsValidIP(address) ? "IPv4:": "URL:"

            if (data.status == null)
                Show_SweetToast("Data saved",header + address + "\n" + "Port:" + port);  
           
        }).fail((data)=>{
            Show_SweetToast("Unable to save the gateway bridge", data.statusText);  
        });

    });

    $("[name=input-IP-bridge]").on("blur keyup paste", function(){

        $(this).val($(this).val().replaceAll(' ', ''));

        if ($(this).val() == "")
            $(this).removeClass("is-valid is-invalid");       
        else
            $(this).addClass("is-valid");

    });

    //********************* Common *********************

    $("[name^=input-name-]").on("keyup",function(){
        $(this).removeClass("is-invalid is-valid")      
    });

    //ip address (also for real gw)
    $("[name^=input-IP]").not("[name=input-IP-bridge]").on("blur keyup",function(){
        
        var value =$(this).val().replaceAll(' ',''); 
        $(this).val(value);

        if (value == "") {
            $(this).removeClass("is-valid is-invalid");
            return;
        }

        ValidationInput($(this),IsValidIP(value));   
            
    });

    //port (also for real gw)
    $("[name^=input-port]").on("blur keyup",function(){

        var value = $(this).val();
       
        if (value.length != 0){

           value = value.replaceAll(' ','').replace(/[^\d,]/g, '');
            $(this).val(value);
            
            if (value.length !=0){
                var ValidPort = value < 65536 && value > 0 ? true : false;
                ValidationInput($(this),ValidPort)             
            }

        } else{
            $(this).removeClass("is-valid is-invalid");
            return;
        }  
    
    });

    //********************* Modal *********************
    $('#modal-location').on('shown.bs.modal', function(event) {
        MapModal.invalidateSize();
    });

    $("#submit-send-payload").on('click',function(){
        
        var ok = CanExecute();
        if (ok) //ok true: SIM off e DEV off
           Show_ErrorSweetToast("Unable send uplink","Simulator is stopped");
        else{

            var address = $(this).parents("#modal-send-data").attr("data-addr");
            
            var data ={
                "id": Devices.get(address).id,
                "mtype": $(this).parents("#modal-send-data").attr("data-mtype"),
                "payload": $(this).parents("#modal-send-data").find("[name=send-payload]").val()
            };
  
            socket.emit("send-uplink",data);
            
        }

        $('#modal-send-data').modal('toggle');

    });

    $("[name=periodicity]").on("blur keyup",function(){

        if($(this).val() == ""){
            $(this).removeClass("is-valid is-invalid");
            return
        }
            
        var value = $(this).val();
        var valid = IsValidNumber(value,-1, 8);     
        
        ValidationInput($(this),valid);

    });

    $("#submit-send-mac-command").on("click",function(){

        var ok = CanExecute();
        if (ok)
            Show_ErrorSweetToast("Unable send MAC Command","Simulator is stopped");
        else{

            var address = $(this).parents("#modal-pingSlotInfoReq").attr("data-addr");
            var valid;

            if($("[name=periodicity]").val() == ""){
                valid = false;
            }else{
                valid = IsValidNumber($("[name=periodicity]").val(),-1, 8);
            }
            
            if (valid){

                var data = {
                    "id": Devices.get(address).id,
                    "cid": "PingSlotInfoReq",
                    "periodicity": Number($(this).parents("#modal-pingSlotInfoReq").find(" [name=periodicity]").val()),
                }
        
                socket.emit("send-MACCommand",data);
            }else{
                Show_ErrorSweetToast("error","Value of periodicity is incorrect. it's between 0 and 7.")
                return;
            }
                
           
        }        
        
        $('#modal-pingSlotInfoReq').modal('toggle');

    });

    $("#submit-new-location").on("click",function(){

        var ok = CanExecute();
        if (ok)
            Show_ErrorSweetToast("Unable change location","Simulator is stopped");
        else{

            var address = $(this).parents("#modal-location").attr("data-addr");

            var latitude = $(this).parents("#modal-location").find("[name=input-latitude]");
            var longitude = $(this).parents("#modal-location").find("[name=input-longitude]");
            var altitude = $(this).parents("#modal-location").find("[name=input-altitude]");

            var validLat = IsValidNumber(Number(latitude.val()),-90.01,90.01);
            var validLng = IsValidNumber(Number(longitude.val()),-180.01,180.01);

            altitude.val(altitude.val() == "" ? 0 : altitude.val());

            ValidationInput(latitude, validLat);
            ValidationInput(longitude, validLng);
            ValidationInput(altitude, true);

            if(!validLat || !validLng){
                Show_iziToast("Values are incorrect","");
                return;
            }

            var data = {
                "id": Devices.get(address).id,
                "latitude":Number(latitude.val()),
                "longitude":Number(longitude.val()),
                "altitude":Number(altitude.val())
            }

            socket.emit("change-location",data,(response)=>{

                var dev = Devices.get(address);

                if (response){

                    dev.info.location.latitude = Number(latitude.val());
                    dev.info.location.longitude = Number(longitude.val());
                    dev.info.location.Altitude = Number(altitude.val());

                    var latlng = L.latLng(Number(latitude.val()),Number(longitude.val()));

                    UpdateMarker(address, address, dev.info.name, latlng, false);

                    Show_iziToast(dev.info.name+" changed location","");

                }else
                    Show_iziToast(dev.info.name+" may be turned off","");
                
            });

        }

        TurnMap = 0;
        $('#modal-location').modal('toggle');

    });

    $("[name=btn-change-payload]").on('click',function(){
        
        var mtype = $("#confirmed-modal").prop("checked") ? ConfirmedData_uplink : UnConfirmedData_uplink;
        var address = $(this).parents("#modal-change-payload").attr("data-addr");

        var ok = CanExecute();
        if (ok)
            Show_ErrorSweetToast("Unable change payload","Simulator is stopped");
        else{
            var dev = Devices.get(address);

            var data ={
                "id": dev.id,
                "mtype": mtype,
                "payload": $("#payload-modal").val()
            };
    
            socket.emit("change-payload",data, (devEUI, ok) => {

                if (ok){
                    var dev = Devices.get(devEUI);

                    dev.info.status.mtype = data.mtype;
                    dev.info.status.payload = data.payload;
                }

            });
    
        }

        $('#modal-change-payload').modal('toggle');
            
    });

});

function Init(){
    
    //list of gateways
    $.ajax({
        url: url+"/api/gateways",
        type:"GET",
        headers:{
            "Access-Control-Allow-Origin":"*"
        }

    }).done((data)=>{

        $("#list-gateways").empty();

        data.forEach(element => {

            Gateways.set(element.info.macAddress, element);

            Add_ItemList_Gateways(element);
   
            AddMarker(element.info.macAddress,element.info.name,
                L.latLng(element.info.location.latitude, element.info.location.longitude),
                true);           
                  
        });

        LoadListHome();

    }).fail((data)=>{
        Show_ErrorSweetToast("Unable to load info of the gateways", data.statusText); 
    });

    //list of devices
    
    $.ajax({
        url: url+"/api/devices",
        type:"GET",
        headers:{
            "Access-Control-Allow-Origin":"*"
        }

    }).done((data)=>{

        $("#list-devices").empty();
        
        data.forEach(element => {

            Devices.set(element.info.devEUI,element)
            
            Add_ItemList_Devices(element);
            
            AddMarker(element.info.devEUI, element.info.name,
                L.latLng(element.info.location.latitude, element.info.location.longitude),
                false);               

        });

        LoadListHome();

    }).fail((data)=>{
        Show_ErrorSweetToast("Unable to load info of the devices", data.statusText); 
    });

}

//********************* Event *********************

function Click_GenerateAddress(selector,bytes){

    var ok = SetData(selector,GenerateAddress(bytes));
    if (ok)
        ValidationInput(selector,true);
    
}

//********************* Notification *********************
function Show_iziToast(title, message){

    iziToast.show({
        title: title,
        message: message,
        position: 'bottomLeft' 
      });

}

function Show_SweetToast(title, message){
    swal(title, message, 'success');   
}

function Show_ErrorSweetToast(title,message){
    swal(title, message, 'error');   
}

//********************* List *********************

function Add_ItemList_Gateways(element){  

    var img ="./img/green_circle.svg";
    if(!element.info.active)
        img ="./img/red_circle.svg";
    
    var type = "Virtual";
    if(element.info.typeGateway)
        type = "Real";

    var item = "<tr data-addr=\""+element.info.macAddress+"\">\
                    <th id=\"state-gw\" scope=\"row\"> \
                        <img src=\""+img+"\">\
                    </th>\
                    <td id=\"name-gw\" class=\"clickable text-orange font-weight-bold font-italic\" >"+element.info.name+"</td>\
                    <td id=\"mac\">"+element.info.macAddress+"</td> \
                    <td id=\"type\">"+type+"</td>\
                </tr>";

    $("#list-gateways").append(item);

}

function Add_ItemList_Devices(element){  

    var img ="./img/green_circle.svg";
    if(!element.info.status.active)
        img ="./img/red_circle.svg";

    var item = "<tr data-addr=\""+element.info.devEUI+"\" class=\"p-5\">\
                    <th id=\"state-dev\" scope=\"row\"> \
                        <img src=\""+img+"\">\
                    </th>\
                    <td id=\"name-dev\" class=\"clickable text-blue font-weight-bold font-italic\" >"+element.info.name+"</td>\
                    <td id=\"devEUI\" > "+element.info.devEUI+"</td> \
                </tr>";

    $("#list-devices").append(item);

}

function ShowList(selector, title, update){

    selector.addClass("active show");
    selector.siblings().removeClass("active show");
    $(".section-header h1").text(title);
    
    if (!update){
        $("a[id *=dev]").parents("li").toggleClass("active");
        $("a[id *=gw]").parents("li").toggleClass("active");  
    }
}

function UpdateList(element, oldAddress, isGw){

    var img = "./img/green_circle.svg";
    var newAddress;

    $("tr[data-addr="+oldAddress+"] [id ^=name-]").text(element.info.name); 

    if(isGw){
        newAddress = element.info.macAddress;

        if(!element.info.active)
            img ="./img/red_circle.svg";
    
        var type = "Virtual";
        if(element.info.typeGateway)
            type = "Real";
        
        $("tr[data-addr="+oldAddress+"]").find("#type").text(type);
        $("tr[data-addr="+oldAddress+"] #mac").text(element.info.macAddress);
        
        $("tr[data-addr="+oldAddress+"]").attr("data-addr",element.info.macAddress);
    }            
    else{
        newAddress = element.info.devEUI;

        if(!element.info.status.active)
            img ="./img/red_circle.svg";

        $("tr[data-addr="+oldAddress+"] #devEUI").text(element.info.devEUI);

        $("tr[data-addr="+oldAddress+"]").attr("data-addr",element.info.devEUI);
    }
        
    $("tr[data-addr="+newAddress+"] [id^=state-]").find("img").attr("src", img);
    
}

function LoadListHome(){
    $("#list-home").empty();

    Devices.forEach(element =>{
        $("#list-home").append("<a href=\"#map-home\" class=\"text-blue list-group-item list-group-item-action\" data-addr=\""+element.info.devEUI+"\">"+element.info.name+"</a>");
    })

    Gateways.forEach(element =>{
        $("#list-home").append("<a href=\"#map-home\" class=\"text-orange list-group-item list-group-item-action\" data-addr=\""+element.info.macAddress+"\">"+element.info.name+"</a>");
    });
    
}

//********************* Validation ********************* 
function IsValidAddress(addr,eui64){

    if (addr == "") return false;

    if (eui64)//addr 64 bit
        return /[0-9A-Fa-f]{16}/.test(addr) && addr.length == LengthEUI64;
    else //addr 16 bit
        return /[0-9A-Fa-f]{8}/.test(addr) && addr.length == LengthDevAddr;
         
}

function ValidationInput(selector, cond){

    if (cond){
        selector.removeClass("is-invalid").addClass("is-valid");
        selector.siblings(".feedback").attr("display","none")
    }
        
    else{
        //error
        selector.removeClass("is-valid").addClass("is-invalid"); 
        selector.siblings(".feedback").attr("display","flex")   
    }

}

function IsValidNumber(value, min, max){//check number
    
    var valid = false;

    if (value > min && value < max) 
        valid = true;

    return valid;
}

function IsValidIP(value){

    var ipFormat = /^(25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)\.(25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)\.(25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)\.(25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)$/;
    
    return ipFormat.test(value);
        
}

function IsValidURL(value){

    //var expression = /[-a-zA-Z0-9@:%._\+~#=]{1,256}\.[a-zA-Z0-9()]{1,6}\b([-a-zA-Z0-9()@:%_\+.~#?&//=]*)?/gi;
    //var regex = new RegExp(expression);

    //return regex.test(value);

    return value == "" ? false : true;
        
}

function IsValidKey(key){
    var regex = new RegExp("^[a-fA-F0-9]+$");
    
    if(key.length != 32 || !regex.test(key))
        return false;
        
    return true;
}

//********************* Functionality ********************* 
function CanExecute(){

    if (!StateSimulator)
        return true;
                      
}

function GenerateAddress(bytes){

    var hexDigits = "0123456789abcdef";
    var Address = "";

    for (var i = 0; i < bytes; i++) {
        Address += hexDigits.charAt(Math.round(Math.random() * 15));
    }

    return Address;
}

//keys
function Check_key(element,event){
    
    var value = $(element).val();

    var valid = IsValidKey(value);

    if (!valid){
        event.preventDefault();
        $(element).val(value.substr(0, value.length));
    }

    ValidationInput($(element),valid);

    if (!valid){
        $(element).siblings(".invalid-feedback").text("The key MUST be 16 characters long. It can contain only letters a-f and number 0-9. ");
    }

}

function SeeKey(selector, selector_image){

    var t = selector.attr('type')

    if (t === "password") {
        selector.attr('type','text');
        selector_image.addClass("seeOFF");
    } else {
        selector.attr('type','password');
        selector_image.removeClass("seeOFF");
    }
}

function GenerateKey(){
    return GenerateAddress(LengthEUI64*2);
}

//********************* Setting generated data ********************* 
function SetData(selector, value){

    if (selector.prop("disabled") == true ||
        selector.prop("readonly") == true)
            return false

    selector.val(value)

    return true;
}

function SetParameters(code, loadDevice, dev){

    socket.emit("get-regional-parameters", code,(data)=>{

        $("#dr-offset-rx1").empty();
        $("#datarate-uplink").empty();
        $("#datarate-rx-2").empty();

        $("#datarate-rx-2").append("<option value=\"-1\"> </option>");
        $("#table-body").empty();

        for (var i = 0; i <= data.maxRX1DROffset; i++)
            $("#dr-offset-rx1").append("<option value=\""+i+"\">"+i+"</option>");

        for (var i = 0; i < data.dataRate.length; i++){

            if (data.dataRate[i] != -1){

                DataRates.push(data.dataRate[i]);
                $("#datarate-uplink").append("<option value=\""+i+"\">"+i+"</option>");
                $("#datarate-rx-2").append("<option value=\""+i+"\">"+i+"</option>");

                var row = "<tr><th scope=\"row\">"+data.dataRate[i]+"</th>";
                row += "<td>"+data.configuration[i]+"</td>";
                row += "<td>"+data.payloadSize[i][0]+"</td>";
                row += "<td>"+data.payloadSize[i][1]+"</td></tr>";
                
                $("#table-body").append(row);
            }
                
        }
        
        frequencyRX2Default = data.frequencyRX2;
        dataRateRX2Default = data.dataRateRX2;
        minFrequency = data.minFrequency;
        maxFrequency = data.maxFrequency;

        var table = "<a href=\"#\" class=\"show-table\" data-toggle=\"modal\" data-target=\"#modal-table\">(Show Table)</a>";
        
        $("#label-freq-rx2").text("Value in Hz. Default value is "+data.frequencyRX2);  
        $("#label-datarate-rx2").html("Default value is "+data.dataRateRX2+". "+table);
        $("#label-datarate-uplink").html(table);

        //da gestire il payload

        if (loadDevice){
            $("#dr-offset-rx1").val(dev.info.configuration.rx1DROffset);
            $("#datarate-rx-2").val(dev.info.rxs[1].dataRate);
            $("#datarate-uplink").val(dev.info.configuration.dataRate);
        }

    });
}

//********************* Map *********************

function Initmap(){

       var osmUrl='http://{s}.tile.openstreetmap.org/{z}/{x}/{y}.png';
       var osmAttrib='&copy; <a href="http://osm.org/copyright">OpenStreetMap</a> contributors'
   
       var osm = new L.TileLayer(osmUrl, {attribution: osmAttrib});
       var osmC= new L.TileLayer(osmUrl, {attribution: osmAttrib});
       var osmHome= new L.TileLayer(osmUrl, {attribution: osmAttrib});
       var osmModal= new L.TileLayer(osmUrl, {attribution: osmAttrib});

       var osmGeocoder = new L.Control.OSMGeocoder({placeholder: 'Search location...'});
       var osmGeocoderDev = new L.Control.OSMGeocoder({placeholder: 'Search location...'});
       var osmGeocoderModal = new L.Control.OSMGeocoder({placeholder: 'Search location...'});

       MapGateway = new L.Map('map-gw').addLayer(osm).setView([CoordDefault, CoordDefault], 10);
       MapGateway.addControl(osmGeocoder);
     
       MapDevice = new L.Map('map-dev').addLayer(osmC).setView([CoordDefault,CoordDefault], 5);
       MapDevice.addControl(osmGeocoderDev);

       MapHome = new L.Map('map-home').addLayer(osmHome).setView([CoordDefault,CoordDefault], 1);
   
       MapModal = new L.Map('map-modal').addLayer(osmModal).setView([CoordDefault,CoordDefault], 5);
       MapModal.addControl(osmGeocoderModal);

       var icon = L.icon({
            iconUrl: './img/orange_marker.svg',
            iconSize: [32, 41],
            iconAnchor:[19,41],
            popupAnchor:[1,-34],
            tooltipAnchor:[16,-28]
        });

       MarkerGateway = L.marker([CoordDefault, CoordDefault],{icon:icon}).addTo(MapGateway);

       icon = L.icon({
            iconUrl: './img/blue_marker.svg',
            iconSize: [32, 41],
            iconAnchor:[19,41],
            popupAnchor:[1,-34],
            tooltipAnchor:[16,-28]
        });

       MarkerDevice = L.marker([CoordDefault, CoordDefault],{icon:icon}).addTo(MapDevice);
       MarkerModal = L.marker([CoordDefault, CoordDefault],{icon:icon}).addTo(MapModal);

}

function GetMap(){

    switch (TurnMap){
        case 0:
            return $("#home");
        case 1:
            return $("#add-dev");
        case 2:
            return $("#add-gw");
        case 3:
            return $("#modal-location")
    }

    return null;
}

function Click_Change_Marker(e){
    
    ChangePositionMarker(-1, e.latlng);
    ChangeCoords(e.latlng);
    
}

function ChangeCoords(latlng){

    var lat,lng;
    var selector = GetMap();
 
    if (latlng == undefined){
        lat = "";
        lng = "";
    }else{
        lat = latlng.lat;
        lng = latlng.lng;
    }
  
    selector.find(" #location").find("#coords [name=input-latitude]").val(lat);
    selector.find(" #location").find("#coords [name=input-longitude]").val(lng);

}

function ChangeView(latlng, zoom){

    switch (TurnMap){

        case 0:
            MapHome.setView(latlng, zoom);
            break;
        case 1:
            MapDevice.setView(latlng, zoom);
            break;
        case 2:
            MapGateway.setView(latlng, zoom);
            break;
        case 3:
            MapModal.setView(latlng, zoom);
            break;
        
    }
}

function CleanMap(){

    $("[name=input-latitude]").val("");
    $("[name=input-latitude]").removeClass("is-valid is-invalid");

    $("[name=input-longitude]").val("");
    $("[name=input-longitude]").removeClass("is-valid is-invalid");

    $("[name=input-altitude]").val("");
    $("[name=input-altitude]").removeClass("is-valid is-invalid");

    ChangePositionMarker(-1,[CoordDefault, CoordDefault]);
    ChangeView([CoordDefault, CoordDefault], 5);
}

function AddMarker(Address, Name, latlng, isGw){
    
    var icon;
    var Marker;

    if(!isGw){

        icon = L.icon({
            iconUrl: './img/blue_marker.svg',
            iconSize: [32, 41],
            iconAnchor:[19,41],
            popupAnchor:[1,-34],
            tooltipAnchor:[16,-28]
        });

        Marker = L.marker(latlng,{icon:icon});
        Marker.bindPopup(GetMenuDevicePopup(Address,Name)).on("popupopen",RegisterEventsPopup);
        Marker.on("click", Click_marker);
        Marker.on("popupclose", FadeCircle);
    }    
    else{

        icon = L.icon({
            iconUrl: './img/orange_marker.svg',
            iconSize: [32, 41],
            iconAnchor:[19,41],
            popupAnchor:[1,-34],
            tooltipAnchor:[16,-28]
        });

        Marker = L.marker(latlng,{icon:icon});
        Marker.bindPopup(GetMenuGatewayPopup(Address, Name, latlng)).on("popupopen",RegisterEventsPopupGw);    
    }
        
    MarkersHome.set(Address,{Address,Marker});
    Marker.addTo(MapHome);
  
}

function UpdateMarker(oldAddress, newAddress, name, latlng, isGw){
    
    var element = MarkersHome.get(oldAddress); 
    if (element != undefined){

        if(isGw)
            element.Marker._popup.setContent(GetMenuGatewayPopup(newAddress, name, latlng))
        else
            element.Marker._popup.setContent(GetMenuDevicePopup(newAddress,name))

        MapHome.closePopup();

        element.Address = newAddress;
        element.Marker.setLatLng(latlng);

    }else
        AddMarker(oldAddress, name, latlng, isGw);       
    
}

function ChangePositionMarker(address, latlng){

    switch (TurnMap){
        case 0:     
            var element = MarkersHome.get(address); 
            if (element != undefined)
                element.Marker.setLatLng(latlng);

            break;

        case 1:
            MarkerDevice.setLatLng(latlng);
            break;

        case 2:
            MarkerGateway.setLatLng(latlng);
            break;

        case 3:
            MarkerModal.setLatLng(latlng);
            break;
    }

}

function RemoveMarker(address){

    MarkersHome.get(address).Marker.removeFrom(MapHome);
    MarkersHome.delete(address);
}

function GetMenuGatewayPopup(address, name, latlng){

    var menu = "<p class=\"text-center m-0 \">"+name+"</p>";
        menu +="<p class=\"m-0 \">Latitude: "+latlng.lat+"</p>";
        menu +="<p class=\"m-0 \">Longitude: "+latlng.lng+"</p>";
        menu +="<div id=\"menu-actions\" data-addr=\""+address+"\" class=\"mh-100 mt-1 overflow-auto list-group list-group-flush\">";
        menu += "<a href=\"#Turn\" class=\"list-group-item item-action p-2\" id=\"turn-gw\"> Toggle On/Off</a>";
    
    return menu;
}

function RegisterEventsPopupGw(){

    $("#turn-gw").on('click',function(){
        
        if (!StateSimulator) {
            Show_ErrorSweetToast("Simulator is stopped","");
            return
        }

        var address = $(this).parent("#menu-actions").attr("data-addr");

        socket.emit("toggleState-gw", Gateways.get(address).id);

        MarkersHome.get(address).Marker.closePopup();
       
    });
}

function GetMenuDevicePopup(address, name){

    var menu = "<p id=\"name-clicked-dev\" class=\"text-center m-0 \">"+name+"</p>";
    menu +="<div id=\"menu-actions\" data-addr=\""+address+"\" class=\"mh-100 mt-1 overflow-auto list-group list-group-flush\">";
    menu += "<a href=\"#Turn\" class=\"list-group-item item-action p-2\" id=\"Turn\"> Toggle On/Off</a>";
    menu += "<a href=\"#send-cdataUp\" class=\"list-group-item item-action p-2\" data-toggle=\"modal\" data-target=\"#modal-send-data\" id=\"send-cdataUp\"> Send ConfirmedDataUp</a>";
    menu += "<a href=\"#send-uncdataUp\" class=\"list-group-item item-action p-2\" data-toggle=\"modal\" data-target=\"#modal-send-data\" id=\"send-uncdataUp\"> Send UnConfirmedDataUp</a>";
    menu += "<a href=\"#DeviceTimeReq\" class=\"list-group-item item-action p-2 mac-command\" data-cmd=\"DeviceTimeReq\" id=\"DeviceTimeReq\"> Send DeviceTimeReq MAC Command</a>";
    menu += "<a href=\"#LinkCheckReq\" class=\"list-group-item item-action p-2 mac-command\" data-cmd=\"LinkCheckReq\" id=\"LinkCheckReq\"> Send LinkCheckReq MAC Command</a>";
    menu += "<a href=\"#PingSlotInfoReq\" class=\"list-group-item item-action p-2 mac-command\" data-toggle=\"modal\" data-target=\"#modal-pingSlotInfoReq\" data-cmd=\"PingSlotInfoReq\" id=\"PingSlotInfoReq\"> Send PingSlotInfoReq MAC Command</a>";
    menu += "<a href=\"#change-location\" class=\"list-group-item item-action p-2 \" data-toggle=\"modal\" data-target=\"#modal-location\" id=\"change-location\"> Change location</a>";
    menu += "<a href=\"#change-payload\" class=\"list-group-item item-action p-2 \" data-toggle=\"modal\" data-target=\"#modal-change-payload\" id=\"change-payload\"> Change payload</a>";
    menu += "</div>";

    return menu;
}

function RegisterEventsPopup(){ 

    $("#Turn").on('click',function(){
        
        var ok = CanExecute();
        if (ok) {
            Show_ErrorSweetToast("Simulator is stopped","");
            return
        }

        var address = $(this).parent("#menu-actions").attr("data-addr");
        
        socket.emit("toggleState-dev", Devices.get(address).id);

        MarkersHome.get(address).Marker.closePopup();
       
    });

    $("#send-cdataUp").on('click',function(){

        var address = $(this).parent("#menu-actions").attr("data-addr");

        $('#modal-send-data').attr("data-addr",address);
        $('#modal-send-data').attr("data-mtype",ConfirmedData_uplink);
        $("#modal-send-data").find("[name=send-payload]").val("");

        MarkersHome.get(address).Marker.closePopup();

    });

    $("#send-uncdataUp").on('click',function(){

        var address = $(this).parent("#menu-actions").attr("data-addr");

        $('#modal-send-data').attr("data-addr",address);
        $('#modal-send-data').attr("data-mtype",UnConfirmedData_uplink);
        $("#modal-send-data").find("[name=send-payload]").val("");

        MarkersHome.get(address).Marker.closePopup();

    });

    $(".mac-command").on('click',function(){

        var address = $(this).parent("#menu-actions").attr("data-addr");
        var cmd = $(this).attr("data-cmd");

        if (cmd == "PingSlotInfoReq"){
            $("[name=periodicity]").removeClass("is-valid is-invalid");
            $("[name=periodicity]").val("");
            $('#modal-pingSlotInfoReq').attr("data-addr",address);
            
        }else{

            var ok = CanExecute();
            if (ok){
                Show_ErrorSweetToast("Unable send MAC Command","Simulator is stopped");
            }else{

                var data = {
                    "id": Devices.get(address).id,
                    "cid":cmd
                }
               
                socket.emit("send-MACCommand",data);
            }

        }

        MarkersHome.get(address).Marker.closePopup();

    });

    $("#change-location").on("click",function(){

        var address = $(this).parent("#menu-actions").attr("data-addr");
        $('#modal-location').attr("data-addr",address);

        $("#modal-location input").removeClass("is-valid is-invalid");
        $("#modal-location input").not("[type=submit]").val("");

        MarkersHome.get(address).Marker.closePopup();
        TurnMap = 3;

    });

    $("#change-payload").on("click",function(){

        var address = $(this).parent("#menu-actions").attr("data-addr");
        $('#modal-change-payload').attr("data-addr",address);

        $("#payload-modal").val("")
        
        MarkersHome.get(address).Marker.closePopup();

    });
  
}

function Click_marker(){
    
    //ottengo l'indirizzo dal popup del marker
    var content = this._popup._content;
    var start = content.indexOf("data-addr=\"")
    var address = content.substring(start+11, start+27);
    //non può essere recuperato dall'evento popupopen 
    //in quanto l'indirizzo non è ancora aggiornato

    var dev = Devices.get(address);
    var range = dev.info.configuration.range;
    var lat = dev.info.location.latitude;
    var lng = dev.info.location.longitude;
    var latlng = new L.latLng(lat,lng);

    DrawRange(latlng, range);
    ChangeView(latlng,11);
}

function FadeCircle(){
    if (Circle != undefined) 
        Circle.removeFrom(MapHome);
}

function DrawRange(latLng, range){

    FadeCircle();

    Circle = new L.circle([latLng.lat, latLng.lng], {
        color: '#0D47A1',
        fillColor: '#0D47A1',
        fillOpacity: 0.5,
        radius: range
    }).addTo(MapHome);
    
}

//********************* Gateway *********************
function CleanInputGateway(){

    $("#add-gw input").not("[type=submit]").val("");
    $("#add-gw input").removeClass("is-valid is-invalid");

    $("#choose-type button").removeClass("active");

    $("[id$=-gw]").prop("checked",false);
    $("#info-gw").addClass("hide");

    $("#div-buttons-gw").removeData("addr");

    CleanMap();

    ChangeStateInputGateway(false, null);

}

function LoadGateway(gw) {

    $("[name=input-name-gw]").val(gw.info.name);
    $("[name=input-MAC-gw]").val(gw.info.macAddress);
    $("#checkbox-active-gw").prop("checked",gw.info.active);

    if (!gw.info.typeGateway){
        $("#virtual-gw").addClass("active");
        $("#info-virtual-gw").removeClass("hide");
        $("#info-real-gw").addClass("hide");
    }else{
        $("#real-gw").addClass("active");
        $("#info-virtual-gw").addClass("hide");
        $("#info-real-gw").removeClass("hide");
    }
    
    $("#info-gw").removeClass("hide");

    //virtual
    $("[name=input-KeepAlive]").val(gw.info.keepAlive);

    //real
    $("[name=input-IP-gw]").val(gw.info.ip);
    $("[name=input-port-gw]").val(gw.info.port);
    
    //map
    var latlng =  L.latLng(gw.info.location.latitude, gw.info.location.longitude); 
    ChangePositionMarker(-1,latlng);
    ChangeView(latlng,8);

    ChangeCoords(latlng);
    $("#add-gw [name=input-altitude]").val(gw.info.location.altitude);
           
    $("#gws").removeClass("show active");
    $("#add-gw").addClass("show active");
    $(".section-header h1").text("Update Gateway");

    ChangeStateInputGateway(true, gw.info.macAddress);
    setTimeout(()=>{
        MapGateway.invalidateSize();
    },300);

}

function ChangeStateInputGateway(value,mac){

    $("#add-gw input").prop({"disabled":value,"readonly":value});
    $("#choose-type button").prop({"disabled":value,"readonly":value});
    if (!value){//new gateway
        $("[name=btn-edit-gw]").hide();
        $("[name=btn-delete-gw]").hide();
        $("[name=btn-save-gw]").show();
    }else{//update gateway
        $("#div-buttons-gw").data("addr",mac);
        $("[name=btn-edit-gw]").show();
        $("[name=btn-delete-gw]").show();
        $("[name=btn-save-gw]").hide();
    }
}

function Click_DeleteGateway(){

    var macAddress = $("#div-buttons-gw").data("addr");
    
    swal({
        title: 'Are you sure?',
        text: 'Once deleted, you will not be able to recover this gateway!',
        icon: 'warning',
        buttons: true,
        dangerMode: true,
        })
        .then((willDelete) => {
        if (willDelete) {

            var gw = Gateways.get(macAddress);
            
            var jsonData = JSON.stringify({
                "id":gw.id
            });

            //ajax
            $.post(url+"/api/del-gateway",jsonData, "json")
            .done((data)=>{
            
                if (data.status){

                    $("tr[data-addr=\""+macAddress+"\"]").remove();

                    Gateways.delete(macAddress);

                    RemoveMarker(macAddress);
                    ShowList($("#gws"),"List gateways",true);
                    Show_SweetToast('Gateway has been deleted!',"");

                }
                else
                    Show_ErrorSweetToast("Error","Gateway didn't deleted. It could be active");                                             

            }).fail((data)=>{    
                Show_ErrorSweetToast("Unable to delete the gateway", data.statusText);
            });            

        } 

        });
    
}

function Click_SaveGateway(){

    var macAddress = $("#div-buttons-gw").data("addr");

    var TypeGateway, valid;
    var id = -1;

    var NameGateway = $("[name=input-name-gw]");
    var MACGateway = $("[name=input-MAC-gw]");
    var isActive = $("#checkbox-active-gw").prop("checked");  
    var KeepAlive = $("[name=input-KeepAlive]");
    var IPGateway = $("[name=input-IP-gw]");
    var PortGateway = $("[name=input-port-gw]");

    if ($("#virtual-gw").hasClass("active")){//virtual

        TypeGateway = false;

        if(KeepAlive.val() == ""){

            valid = true;
            KeepAlive.val(KeepAliveDefault);

        } else
            valid = KeepAlive.val() <= 0 ? false : true; 

        ValidationInput(KeepAlive, valid);
        IPGateway.val("");
        PortGateway.val("");


    }else if ($("#real-gw").hasClass("active")){//real

        TypeGateway = true;
        
        valid = IsValidIP(IPGateway.val());
        ValidationInput(IPGateway, valid);

        var validPort = IsValidNumber(PortGateway.val(),-1,65536);
        ValidationInput(IPGateway, validPort);

        valid = validPort ? valid : false;

        KeepAlive.val("");
    }

    //map
    var selector = $("#div-buttons-gw").siblings("#location").find(" #coords");
    var latitude = selector.find(" [name=input-latitude]");
    var longitude = selector.find(" [name=input-longitude]");
    var altitude = selector.find(" [name=input-altitude]");

    latitude.val(latitude.val()=="" ? 0 : latitude.val());
    longitude.val(longitude.val()=="" ? 0 : longitude.val());
    altitude.val(altitude.val()=="" ? 0 : altitude.val());

    var validLat = IsValidNumber(Number(latitude.val()),-90.01, 90.01);
    var validLng = IsValidNumber(Number(longitude.val()),-180.01, 180.01);

    var location = {
        "latitude":Number(latitude.val()),
        "longitude": Number(longitude.val()),
        "altitude": Number(altitude.val())
    }

    //validation
    var validNameGateway = NameGateway.val() == "" ? false : true;  
    var validMACGateway = IsValidAddress(MACGateway.val(),true) ? true : false; 

    if(!validMACGateway || !validNameGateway || !valid || !validLat || !validLng ){//Error
       
        Show_ErrorSweetToast("Error: values are incorrect","");

        ValidationInput(NameGateway, validNameGateway);   
        ValidationInput(MACGateway, validMACGateway);
        ValidationInput(latitude, validLat);
        ValidationInput(longitude, validLng);
        ValidationInput(altitude, true); 

        return;
    }

    if (macAddress != undefined && macAddress != "") // update gateway
        id = Gateways.get(macAddress).id;  

    var gw = {
        "id": id,
        "info":{
            "active": isActive,
            "name" : NameGateway.val(),
            "macAddress" : MACGateway.val().toLowerCase(),
            "keepAlive": Number(KeepAlive.val()),
            "typeGateway":TypeGateway,
            "ip":IPGateway.val(),
            "port": PortGateway.val(),
            "location":location
        }
    };

    //file JSON
    var jsonData= JSON.stringify(gw);

    if (macAddress == undefined || macAddress == ""){//new gateway
  
        $.post(url + "/api/add-gateway",jsonData, "json")
        .done((data)=>{

            switch (data.code){
                case 0:

                    gw.id = data.id;

                    Gateways.set(gw.info.macAddress,gw);

                    AddMarker(gw.info.macAddress,gw.info.name,
                        L.latLng(gw.info.location.latitude,gw.info.location.longitude),
                        true);
            
                    Add_ItemList_Gateways(gw);

                    CleanInputGateway();

                    ShowList($("#gws"),"List gateways",false);
                    Show_SweetToast("Gateway Saved","");

                    return;

                case 1:// same name

                    NameGateway.addClass("is-invalid");
                    NameGateway.siblings(".invalid-feedback").text(data.status);

                    break;

                case 2:

                    NameGateway.addClass("is-invalid");
                    MACGateway.addClass("is-invalid");
                    MACGateway.siblings(".invalid-feedback").text(data.status);

                    break;   
                    
                case 4:
                    Show_ErrorSweetToast("Error",data.status)
                        
            }  

            Show_ErrorSweetToast("Error",data.status);
            
        }).fail((data)=>{    
            Show_ErrorSweetToast("Unable to save the gateway", data.statusText);
        });

    } else{//update gateway

        $.post(url + "/api/up-gateway",jsonData, "json")
        .done((data)=>{

            switch (data.code){
                case 0:

                    Gateways.delete(macAddress);
                    Gateways.set(gw.info.macAddress, gw);

                    var latlng = L.latLng(gw.info.location.latitude, gw.info.location.longitude);      
                    UpdateMarker(macAddress, gw.info.macAddress, gw.info.name, latlng, true);
                
                    UpdateList(gw, macAddress, true);
                    ShowList($("#gws"),"List gateways",true);

                    Show_SweetToast("Gateway updated","");
    
                    CleanInputGateway();

                    return;

                case 1:// same name

                    NameGateway.addClass("is-invalid");
                    NameGateway.siblings(".invalid-feedback").text(data.status);

                    break;

                case 2:

                    NameGateway.addClass("is-invalid");
                    MACGateway.addClass("is-invalid");
                    MACGateway.siblings(".invalid-feedback").text(data.status);

                    break;

                case 5:
                    Show_ErrorSweetToast("Error",data.status); 
                    return;

            }

            Show_ErrorSweetToast("Error",data.status);
                      
        }).fail((data)=>{    
            Show_ErrorSweetToast("Unable to update the gateway", data.statusText);   
        });
    }
    
}

//********************* Device *********************
function CleanActivation(){
    //check già gestita

    $('#activation input').prop({"disabled":false,"readonly":false});
    
    $('[name=btn-watch] > img').removeClass("seeOFF");
    $('[name^=input-key] ').attr("type","password");

}

function CleanInputDevice(){

    $("#add-dev input").not("[type=submit]").val("");
    $("#add-dev input").removeClass("is-valid is-invalid");
    $("#add-dev [type=checkbox]").prop("checked",false);

    $("#region").val(-1);

    $("select").removeClass("is-valid is-invalid");
    $("#table-body").empty();

    $("#dr-offset-rx1").empty();
    $("#datarate-uplink").empty();
    $("#datarate-rx-2").empty();

    //Activation
    CleanActivation();

    $("#div-buttons-dev").removeData("addr");

    $("#textarea-payload").val("");

    //location
    CleanMap();

    ChangeStateInputDevice(false,null);

    $("#add-dev *").removeClass("active show");
    $("#general-tab").addClass("active");
    $("#general").addClass("active show");
}

function ChangeStateActivation(otaa){

    if (otaa){//otaa supported

        $("[name=input-key-app]").prop({"disabled":false,"readonly":false});

        $("[name=input-devAddr]").prop({"disabled":true,"readonly":true});
        $("[name^=input-key]").not("[name=input-key-app]").prop({"disabled":true,"readonly":true});

    } else{

        $("[name=input-key-app]").prop({"disabled":true,"readonly":true});

        $("[name=input-devAddr]").prop({"disabled":false,"readonly":false});
        $("[name^=input-key]").not("[name=input-key-app]").prop({"disabled":false,"readonly":false});
    
    }
}

function ChangeStateInputDevice(value,eui){

    $("#add-dev input").prop({"disabled":value,"readonly":value});
    $("#add-dev select").prop({"disabled":value,"readonly":value});

    $("[name=input-mtype]").prop({"disabled":value,"readonly":value});
    $("#textarea-payload").prop({"disabled":value,"readonly":value});

    if (!value){//new device

        var otaa = $("#checkbox-otaa-dev").prop("checked");
        ChangeStateActivation(otaa);

        var ClassB =$("#classB-dev").prop("checked");
        var ClassC =$("#classC-dev").prop("checked");

        if(ClassB)
            $("#classB-dev").prop({"disabled":true,"readonly":true});
        else if (ClassC)
            $("#classC-dev").prop({"disabled":true,"readonly":true});

        var disableCntDl=$("[name=input-validate-counter]").prop("checked");  
        if(disableCntDl)
            $("[name=input-fcnt-downlink]").prop({"disabled":true,"readonly":true});

        //buttons       
        $("[name=btn-edit-dev]").hide();
        $("[name=btn-edit-dev]").hide();
        $("[name=btn-delete-dev]").hide();
        $("[name=btn-save-dev]").show();

    }else{//update device

        $("#div-buttons-dev").data("addr",eui);

        $("[name=btn-edit-dev]").show();
        $("[name=btn-delete-dev]").show();
        $("[name=btn-save-dev]").hide();

    }

}

function LoadDevice(dev){

    //general
    $("[name=checkbox-active-dev]").prop("checked",dev.info.status.active);
    $("[name=input-name-dev]").val(dev.info.name);
    $("[name=input-devEUI]").val(dev.info.devEUI);
    $("#region").val(dev.info.configuration.region);
    
    SetParameters(dev.info.configuration.region, true,dev);

    //activation
    $("#checkbox-otaa-dev").prop("checked",dev.info.configuration.supportedOtaa);
    if(dev.info.configuration.supportedOtaa)
        $("[name=input-key-app]").val(dev.info.appKey);    
    else{
        $("[name=input-devAddr]").val(dev.info.devAddr);
        $("[name=input-key-nwkS]").val(dev.info.nwkSKey);
        $("[name=input-key-appS").val(dev.info.appSKey);
    }

    //class A
    $("[name=input-rx-1-delay]").val(dev.info.rxs[0].delay);
    $("[name=input-rx-1-duration]").val(dev.info.rxs[0].durationOpen);   
    $("[name=input-rx-2-delay]").val(dev.info.rxs[1].delay);
    $("[name=input-rx-2-duration]").val(dev.info.rxs[1].durationOpen);
    $("[name=input-frequency-rx-2]").val(dev.info.rxs[1].channel.freqDownlink);   
    $("[name=input-ackTimeout]").val(dev.info.configuration.ackTimeout);
    $("#dr-offset-rx1").val(dev.info.configuration.rx1DROffset);

    //ClassB,C
    $("#classB-dev").prop("checked", dev.info.configuration.supportedClassB);
    $("#classC-dev").prop("checked", dev.info.configuration.supportedClassC);

    //frame settings
    
    $("[name=input-fport]").val(dev.info.status.infoUplink.fport);
    $("[name=input-retransmission]").val(dev.info.configuration.nbRetransmission)
    $("[name=input-fcnt]").val(dev.info.status.infoUplink.fcnt);
    $("#datarate-uplink").val(dev.info.configuration.dataRate);

    $("[name=input-validate-counter]").prop("checked",dev.info.configuration.disablefcntDown);  
    if (!dev.info.configuration.disablefcntDown)
        $("[name=input-fcnt-downlink]").val(dev.info.status.fcntDown);

    //features
    $("[name=input-ADR]").prop("checked", dev.info.configuration.supportedADR);
    $("[name=input-range]").val(dev.info.configuration.range);
    
    //location
    var latlng =  L.latLng(dev.info.location.latitude, dev.info.location.longitude); 
    ChangePositionMarker(-1,latlng);
    ChangeView(latlng, 8);

    ChangeCoords(latlng);
    $("#add-dev [name=input-altitude]").val(dev.info.location.altitude);
    

    //payload
    $("[name=input-sendInterval]").val(dev.info.configuration.sendInterval);

    if(dev.info.status.mtype == ConfirmedData_uplink){
        $("#confirmed").prop("checked",true);
        $("#unconfirmed").prop("checked",false);
    }else{
        $("#confirmed").prop("checked",false);
        $("#unconfirmed").prop("checked",true);
    }

    $("#fragments").prop("checked",dev.info.configuration.supportedFragment);
    $("#truncates").prop("checked",!dev.info.configuration.supportedFragment);
  
    $("#textarea-payload").val(dev.info.status.payload);

    ChangeStateInputDevice(true,dev.info.devEUI);


    $("#devs").removeClass("show active");
    $("#add-dev").addClass("show active");
    $(".section-header h1").text("Update Device");

}

function Click_DeleteDevice(){

    swal({
        title: 'Are you sure?',
        text: 'Once deleted, you will not be able to recover this device!',
        icon: 'warning',
        buttons: true,
        dangerMode: true,
        })
        .then((willDelete) => {
        if (willDelete) {

            var devEUI = $("#div-buttons-dev").data("addr");

            var jsonData = JSON.stringify({
                "id" : Devices.get(devEUI).id
            });

            //ajax
            $.post(url+"/api/del-device",jsonData, "json")
            .done((data)=>{
        
                if (data.status){
        
                    $("tr[data-addr=\""+devEUI+"\"]").remove();

                    Devices.delete(devEUI);

                    RemoveMarker(devEUI);
                    ShowList($("#devs"),"List devices",true);
                    Show_SweetToast("Device Deleted","");

                }
                else
                    Show_ErrorSweetToast("Error","Device didn't deleted. It could be active");

            }).fail((data)=>{    
                Show_ErrorSweetToast("Unable to delete the device", data.statusText);
            });
        }
    });   

}

function Click_SaveDevice(){

    var validation;

    //******************* general ***************************

    var active = $("[name=checkbox-active-dev]").prop("checked");
    var name = $("[name=input-name-dev]");
    var devEUI = $("[name=input-devEUI]");
    var region = $("#region");

    var validNameDevice = name.val() == "" ? false : true;  
    var validdevEUI = IsValidAddress(devEUI.val(),true);
    var validregion = region.val() == -1 ? false: true;
    
    validation = validdevEUI && validNameDevice && validregion;

    //******************* activation ***************************

    var supportedOtaa = $("#checkbox-otaa-dev").prop("checked");
    var devAddr = $("[name=input-devAddr]");
    var nwkSKey = $("[name=input-key-nwkS]");
    var appSKey = $("[name=input-key-appS");
    var appKey = $("[name=input-key-app]");

    var valuedevAddr = "";
    var valuenwkSKey = "";
    var valueappSKey = "";

    if (supportedOtaa){

        ValidationInput(appKey, IsValidKey(appKey.val()));
        validation = validation && IsValidKey(appKey.val());

    }else{

        var validdevAddr = IsValidAddress(devAddr.val(), false);       
        var validnwkSKey = IsValidKey(nwkSKey.val());        
        var validappSKey = IsValidKey(appSKey.val());

        valuedevAddr = devAddr.val();
        valuenwkSKey = nwkSKey.val();
        valueappSKey = appSKey.val(); 

        ValidationInput(devAddr, validdevAddr);
        ValidationInput(nwkSKey, validnwkSKey);
        ValidationInput(appSKey, validappSKey);

        validation = validation && validdevAddr && validnwkSKey && validappSKey;
    }
    
    //******************* class A ***************************

    var delayRX1 = $("[name=input-rx-1-delay]");
    var durationRX1 = $("[name=input-rx-1-duration]");
    var DROffsetRX1 = $("#dr-offset-rx1");
    var delayRX2 = $("[name=input-rx-2-delay]");
    var durationRX2 = $("[name=input-rx-2-duration]");
    var frequencyRX2 = $("[name=input-frequency-rx-2]");
    var dataRateRX2 = $("#datarate-rx-2");
    var ackTimeout = $("[name=input-ackTimeout]");

    delayRX1.val(delayRX1.val() == "" ? DelayDefault : delayRX1.val());
    var validDelayRX1 = IsValidNumber(delayRX1.val(),0,Infinity);

    durationRX1.val(durationRX1.val() == "" ? DurationDefault : durationRX1.val());
    var validDurationRX1 = IsValidNumber(durationRX1.val(),0,Infinity);
    
    delayRX2.val(delayRX2.val() == "" ? DelayDefault : delayRX2.val());
    var validDelayRX2 = IsValidNumber(delayRX2.val(),0,Infinity);

    durationRX2.val(durationRX2.val() == "" ? DurationDefault : durationRX2.val());
    var validDurationRX2 = IsValidNumber(durationRX2.val(),0,Infinity);
 
    frequencyRX2.val(frequencyRX2.val() == "" ? frequencyRX2Default : frequencyRX2.val());
    var validfrequencyRX2 = IsValidNumber(frequencyRX2.val(),minFrequency-0.01,maxFrequency+0.01);

    dataRateRX2.val(dataRateRX2.val() == -1 ? dataRateRX2Default : dataRateRX2.val());
    
    ackTimeout.val(ackTimeout.val() == "" ? ACKTimeoutDefault : ackTimeout.val());
    var validackTimeout= IsValidNumber(ackTimeout.val(),-1,4);

    validation = validation && validDelayRX1 && validDelayRX2 && validDurationRX1;
    validation = validation && validDurationRX2 && validfrequencyRX2 && validackTimeout;

    //******************* Class B e C ***************************

    var isClassBactive = $("#classB-dev").prop("checked");
    var isClassCactive = $("#classC-dev").prop("checked");

    //******************* frame's settings ***************************

    var datarate = $("#datarate-uplink");
    var fport = $("[name=input-fport]");
    var retransmission = $("[name=input-retransmission]")
    var Fcnt = $("[name=input-fcnt]");
    var disablefcntDown = $("[name=input-validate-counter]").prop("checked");
    var fcntDown = $("[name=input-fcnt-downlink]");

    var validFport = false;
    if(fport.val() != "")
        validFport = IsValidNumber(fport.val(),0,224);

    var validReply = false;
    if(retransmission.val() != "")
        validReply = IsValidNumber(retransmission.val(),-1,Infinity);
    
    var validFcnt = false;
    if(Fcnt.val() != "")
        validFcnt = IsValidNumber(Fcnt.val(),-1,MaxValueCounter+1);
    
    var validFcntDown = true;
    if(!disablefcntDown){
    
        if(fcntDown.val() == "")
            validFcntDown = false;
        else
            validFcntDown = IsValidNumber(fcntDown.val(),-1,MaxValueCounter+1);
    } 

    validation = validation && validFport && validReply && validFcnt && validFcntDown;

    //******************* features ***************************

    var supportedADR = $("[name=input-ADR]").prop("checked");
    var range = $("[name=input-range]");

    $("[name=input-range]").val($("[name=input-range]").val() == "" ? rangeDefault : $("[name=input-range]").val())
    var validrange = IsValidNumber(range.val(),0,Infinity);

    validation = validation && validrange;

    //******************* location ***************************

    var selector = GetMap();
    var latitude = selector.find("#coords").find(" [name=input-latitude]");
    var longitude = selector.find("#coords").find(" [name=input-longitude]");
    var altitude = selector.find("#coords").find(" [name=input-altitude]");

    var validLat = IsValidNumber(Number(latitude.val()),-90.01,90.01);
    var validLng = IsValidNumber(Number(longitude.val()),-180.01,180.01);
    altitude.val(altitude.val() == "" ? altitude.val(0): altitude.val());

    var location = {
        "latitude":Number(latitude.val()),
        "longitude": Number(longitude.val()),
        "altitude":Number(altitude.val())
    }

    validation = validation && validLat && validLng;
 
    //******************* payload ***************************

    var mtype = $("#confirmed").prop("checked") ? ConfirmedData_uplink : UnConfirmedData_uplink; //true Confirmed
    var upInterval = $("[name=input-sendInterval]");
    var payload = $("#textarea-payload").val();

    upInterval.val(upInterval.val() == "" ? UplinkIntervalDefault : upInterval.val());
    var validInterval = IsValidNumber(upInterval.val(),-1,Infinity);

    validation = validation && validInterval;
    
    if (!validation){
        Show_ErrorSweetToast("Error","Values are incorrect");

        ValidationInput(name, validNameDevice);
        ValidationInput(devEUI, validdevEUI);
        ValidationInput(region, validregion);

        ValidationInput(delayRX1,validDelayRX1);
        ValidationInput(durationRX1,validDurationRX1);
        ValidationInput(DROffsetRX1,true);
        ValidationInput(delayRX2,validDelayRX2);
        ValidationInput(durationRX2,validDurationRX2);
        ValidationInput(dataRateRX2,true);
        ValidationInput(frequencyRX2,validfrequencyRX2);
        ValidationInput(ackTimeout,validackTimeout);

        ValidationInput(fport,validFport);
        ValidationInput(retransmission,validReply);
        ValidationInput(Fcnt,validFcnt);
        ValidationInput(fcntDown,validFcntDown);

        ValidationInput(range,validrange);

        ValidationInput(latitude, validLat);
        ValidationInput(longitude, validLng);
        ValidationInput(altitude, true);

        ValidationInput(upInterval,validInterval);
        
        return;
    }

    var eui = $("#div-buttons-dev").data("addr");
    var id = -1;

    if (eui != undefined && eui != "")// update device       
        id = Devices.get(eui).id;
    
    //JSON creation, lo invio tutto sotto forma di stringa (?)
    var dev = {
        "id":id,
        "info":{
            "name" : name.val(),
            "devEUI" : devEUI.val().toLowerCase(),
            "appKey": appKey.val().toLowerCase(),
            "devAddr": valuedevAddr.toLowerCase(),
            "nwkSKey": valuenwkSKey.toLowerCase(),
            "appSKey": valueappSKey.toLowerCase(),
            "location":location,
            "status":{
                "active": active,
                "infoUplink":{
                    "fport": Number(fport.val()),
                    "fcnt": Number(Fcnt.val()),
                },
                "mtype": mtype,
                "payload":payload,
                "fcntDown": Number(fcntDown.val())
            },
            "configuration":{
                "region":Number(region.val()),
                "ackTimeout":Number(ackTimeout.val()),
                "rx1DROffset":Number(DROffsetRX1.val()),
                "supportedADR":supportedADR,
                "supportedOtaa":supportedOtaa,
                "supportedFragment":$("#fragments").prop("checked"),
                "supportedClassB":isClassBactive,
                "supportedClassC":isClassCactive,
                "range":Number(range.val()),
                "dataRate": Number(datarate.val()),
                "disableFCntDown":disablefcntDown,
                "sendInterval":Number(upInterval.val()),
                "nbRetransmission":Number(retransmission.val()),
            },
            "rxs":[
                {
                    "delay":Number(delayRX1.val()),
                    "durationOpen":Number(durationRX1.val())
                },
                {
                    "channel":{
                        "active":true,
                        "freqDownlink": Number(frequencyRX2.val())
                    },
                    "delay":Number(delayRX2.val()),
                    "durationOpen":Number(durationRX2.val()),
                    "dataRate":Number(dataRateRX2.val())
                }
            ]
        }    
    };
    
    var jsonData = JSON.stringify(dev);

    if (eui == undefined || eui == ""){//new device 

        $.post(url + "/api/add-device",jsonData, "json")
        .done((data)=>{

            switch (data.code){
                case 0: //OK

                    dev.id = data.id;

                    AddMarker(dev.info.devEUI,dev.info.name,
                        L.latLng(dev.info.location.latitude, dev.info.location.longitude),
                        false);
    
                    Devices.set(dev.info.devEUI, dev);
    
                    Add_ItemList_Devices(dev);//ui
                    ShowList($("#devs"),"List devices",false);            

                    CleanInputDevice();

                    Show_SweetToast("New Device saved","");
                            
                    return;

                case 1:// same name

                    $("[name=input-name-dev]").addClass("is-invalid");
                    $("[name=input-devEUI]").siblings(".invalid-feedback").text(data.status);

                    return;

                case 2:

                    $("[name=input-name-dev]").addClass("is-invalid");
                    $("[name=input-devEUI]").addClass("is-invalid");
                    $("[name=input-devEUI]").siblings(".invalid-feedback").text(data.status);

                    return;
                   
            }
                 
        }).fail((data)=>{    
            Show_ErrorSweetToast("Unable to save the device", data.statusText);
        });

    } else{//update device

        $.post(url + "/api/up-device",jsonData, "json")
        .done((data)=>{

            switch (data.code){
                case 0:

                    Devices.delete(eui);
                    Devices.set(dev.info.devEUI, dev);                 

                    var latlng = L.latLng(dev.info.location.latitude, dev.info.location.longitude);
                    UpdateMarker(eui, dev.info.devEUI, dev.info.name, latlng, false);
                    
                    UpdateList(dev, eui, false);
                    ShowList($("#devs"),"List devices",true);

                    Show_SweetToast("Device updated","");

                    CleanInputDevice();

                    return;

                case 1:// same name

                $("[name=input-name-dev]").addClass("is-invalid");
                    break;

                case 2:

                    $("[name=input-devEUI]").addClass("is-invalid");
                    break;

                case 3:
                    Show_ErrorSweetToast("Error",data.status); 
                    return;

            }   
            Show_ErrorSweetToast("Error",data.status);
            
        }).fail((data)=>{    
            Show_ErrorSweetToast("Unable to update the device", data.statusText);  
        });
    }
    
}

//********************* Common *********************
function Click_Edit(element, FlagGw){
 
    var addr = $("#div-buttons-dev").data("addr");

    if(FlagGw)
        addr = $("#div-buttons-gw").data("addr");   

    $(element).hide();   
    
    if (FlagGw) //Edit gateway
        ChangeStateInputGateway(false,null);        
    else
        ChangeStateInputDevice(false,null);

    $(element).siblings("button").show();
   
}
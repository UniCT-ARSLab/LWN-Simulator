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
const RangeDefault = 10000;
const MaxValueCounter = 16384;
const UnConfirmedData_uplink ="UnConfirmedDataUp"
const ConfirmedData_uplink ="ConfirmedDataUp"

var PairsEUI64 = 8;
var PairsDevAddr = 4;

var StateSimulator = false;//true in running

//checks
var DataRates = [];
var MinFrequency = 100;
var Maxfrequency = 100;
var TablePayload = [];
var TablePayloadDT = [];
var FrequencyRX2Default = 0;
var DatarateRX2Default = 0;

//maps
var MapGateway;
var MapDevice;
var MapHome;
var MapModal;

var MarkerGateway = {};
var MarkerDevice = {};
var MarkersHome = [];
var MarkerModal = {};
var Circle;

var TurnMap = 0;

var Gateways = [];
var Devices = [];
var DeviceRunning = [];
var GatewaysRunning = [];
//socket
var socket = io('http://127.0.0.1:8000');

$(document).ready(function(){

    Init();
    Initmap();
    setTimeout(() => {
        MapGateway.invalidateSize() 
    }, 500)

    MapGateway.on('click',Change_Marker);
    MapDevice.on('click',Change_Marker);
    MapModal.on('click',Change_Marker);

    // ********************** socket event *********************

    socket.on('connect',()=>{
        console.log("socket connessa");
    })

    socket.on('console-sim',(data)=>{

        var row = "<p class=\"text-break text-start bg-secondary m-0\">"+data.Msg+"</p>";
        $('#console-body').append(row);

        $('#console-body').animate({
            scrollTop: $('#console-body').get(0).scrollHeight
        }, 0);

    });

    socket.on('console-error',(data)=>{
      
        var row = "<p class=\"text-break text-white bg-danger m-0\">"+data.Msg+"</p>";             
        $('#console-body').append(row);

        $('#console-body').animate({
            scrollTop: $('#console-body').get(0).scrollHeight
        }, 0);

    });

    socket.on('log-dev',(data)=>{

        var classesP = $("[name=\""+data.Name+"\"]").attr("class");//p
        
        $("span[data-name=\""+data.Name+"\"]").attr("class");

        var row;

        if (classesP == undefined)
            row = "<p class=\"text-break text-start text-info clickable me-1 mb-0\" name=\""+data.Name+"\" data-name=\""+data.Name+"\">"+data.Msg+"</p>";
        else
            row = "<p class=\""+classesP+"\" name=\""+data.Name+"\" data-name=\""+data.Name+"\">"+data.Msg+"</p>";
    
        $('#console-body').append(row);

        $('#console-body').animate({
            scrollTop: $('#console-body').get(0).scrollHeight
        }, 0);

    });

    socket.on('log-gw',(data)=>{ 

        var valueName = "gw-"+data.Name;     
        var classesP = $("[name="+valueName+"]").attr("class");//p
        var row;

        if (classesP == undefined)
            row = "<p class=\"text-break clickable text-start text-warning me-1 mb-0\" data-name=\""+valueName+"\" name=\""+valueName+"\">"+data.Msg+"</p>";
        else
            row = "<p class=\""+classesP+"\" name="+valueName+" data-name=\""+valueName+"\">"+data.Msg+"</p>";
    
        $('#console-body').append(row);

        $('#console-body').animate({
            scrollTop: $('#console-body').get(0).scrollHeight
        }, 0);

    });

    socket.on('save-status',(data)=>{

        var index = Devices.findIndex((x) => x.Info.DevEUI == data.DevEUI);

        Devices[index].Info.DevAddr = data.DevAddr;
        Devices[index].Info.NwkSKey = data.NwkSKey;
        Devices[index].Info.AppSKey = data.AppSKey;
        Devices[index].Info.Status.FCntDown = data.FCntDown;
        Devices[index].Info.Status.InfoUplink.FCnt = data.FCnt;

    });

    socket.on('response-command',(data)=>{
        Show_iziToast(data,"");
    });

    // ********************** nav bar *********************

    $("a > i.btn-play").on("click",function(){
        
        if (!socket.connected){
            Show_ErrorSweetToast("Socket not connected","");
            return;
        }

        if (StateSimulator){
            Show_ErrorSweetToast("Simulator already run","");
            return;
        }

        $(this).parent("a").addClass("beep");
        $("#state").attr("src","img/yellow_circle.png");

        $.ajax({
            url:"http://127.0.0.1:8000/api/start",
            type:"GET",
            headers:{
                "Access-Control-Allow-Origin":"*"
            }
        }).done((data)=>{
            
            if (data){
                
                StateSimulator = true;
                DeviceRunning = [];
                Devices.forEach(element =>{
                    DeviceRunning.push({"DevEUI":element.Info.DevEUI,
                                        "Active": element.Info.Status.Active
                                        });
                });

                GatewaysRunning = [];
                Gateways.forEach(element =>{
                    GatewaysRunning.push({"MACAddress":element.Info.MACAddress,
                                        "Active": element.Info.Active
                                        });
                });

                Show_iziToast("Simulator started","");
                $("#state").attr("src","img/green_circle.png");
            }               
            else{
                Show_ErrorSweetToast("Error","Simulator didn't started");
                $("#state").attr("src","img/red_circle.png");
            }
                

                  
        }).fail((data)=>{

            $("#state").attr("src","img/red_circle.png");
            Show_ErrorSweetToast("Error",data.statusText); 
            
        }).always(()=>{
            $(this).parent("a").removeClass("beep");
        });

    });

    $("a > i.btn-stop").on("click",function(){

        if(!StateSimulator){
            Show_ErrorSweetToast("Simulator already stop","");
            return;
        }
        
        StateSimulator = false;

        $(this).parent("a").addClass("beep");
        $("#state").attr("src","img/yellow_circle.png");

        $.get("http://127.0.0.1:8000/api/stop",{
        
        }).done((data)=>{
        
            if (data){

                $("#state").attr("src","img/red_circle.png");
                Show_iziToast("Simulator stopped","");

            }
            else{
                $("#state").attr("src","img/green_circle.png");
            }
            

        }).fail(()=>{

            $("#state").attr("src","img/green_circle.png");

        }).always(()=>{

            $(this).parent("a").removeClass("beep");
        });

    });
    
    // ********************** sidebar *********************

    $("#sidebar a").on("click", function () {
    
        $(".main-content > div").removeClass("show active");
        $($(this).data("tab")).addClass("show active");

        if (this.id == "home-tab"){
            TurnMap = 0;
            $(".section-header h1").text("LWN Simulator");

            LoadListHome();
        }
                            
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
        var index = MarkersHome.findIndex((x) => x.Address == address);
        var indexDev = Devices.findIndex((x) => x.Info.DevEUI == address);
        
        ChangeView(MarkersHome[index].Marker.getLatLng())
        MarkersHome[index].Marker.openPopup();

        if(indexDev != -1)
            DrawRange(MarkersHome[index].Marker.getLatLng(), Devices[indexDev].Info.Configuration.Range);
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

        ChangeView(latlng);
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

        ChangeView(latlng);
        ChangePositionMarker(-1,latlng);
    });

    // ********************** sidebar/dropdown: list devices *********************
    
    //click item list
    $("#list-devices").on("click","a",function(){ 

        var address = $(this).attr("data-addr");
        var index = Devices.findIndex((x) => x.Info.DevEUI == address);

        CleanInputDevice();
        LoadDevice(Devices[index]);
        $("#header-sidebar-dev h4").text($(this).text());
    });
    
    // ********************** sidebar/dropdown: add new device *********************

    $("#location-tab").on("click",function(){
        setTimeout(()=>{
            MapDevice.invalidateSize();
        },300);
        
    });

    $("[name=input-devEUI]").on('blur keyup',function(e){
        var valid = KeyUp_ValidAddress(this, e, PairsEUI64*2);   

        ValidationInput($(this),valid); 

        if ($(this).val().length == 0)
            $(this).removeClass("is-valid is-invalid");   
    });

    //generate DevEUI
    $('[name=btn-new-devEUI]').on('click',function(){
        Click_GenerateAddress($("[name=input-devEUI]"), PairsEUI64);
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
    $('[name=input-devAddr]').on('blur keyup',function(e){
        var valid = KeyUp_ValidAddress(this, e, PairsDevAddr*2);   
        
        ValidationInput($(this),valid); 
        
        if ($(this).val().length ==0)
            $(this).removeClass("is-valid is-invalid");
    });

    //generate DevAddr
    $('[name=btn-new-devAddr]').on('click',function(){
        Click_GenerateAddress($("[name=input-devAddr]"),PairsDevAddr);
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

        var valid = IsValidNumber($(this).val(),MinFrequency, Maxfrequency);
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
        var valid = IsValidNumber(this,0, Infinity);
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
    $("#list-gateways").on("click","a",function(){
        
        var MACAddress = $(this).attr("data-addr");
        var index = Gateways.findIndex((x) => x.Info.MACAddress == MACAddress);//to modify existed gateway
        
        Load_Gateway(Gateways[index]);
    });

    // ********************** sidebar/dropdown: add new gateway *********************

    $("#choose-type label").on("click", function(){
        setTimeout(()=>{
            MapGateway.invalidateSize();
        },300);
    });

    //input MAC 
    $("[name=input-MAC-gw]").on("blur keyup",function(e){

        var valid = KeyUp_ValidAddress(this, e, PairsEUI64*2);   

        ValidationInput($(this),valid); 

        if ($(this).val().length ==0)
            $(this).removeClass("is-valid is-invalid");
    });

    //generate new mac address
    $("[name=btn-new-MACAddress]").on("click",function(){
        Click_GenerateAddress($(this).siblings("[name=input-MAC-gw]"),PairsEUI64);
    });

    $("#choose-type label").on('click',function(){
        $("#info-gw").removeClass("hide");
        
        if ($(this).children("input").attr("id") == "virtual-gw"){
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
            url: "http://127.0.0.1:8000/api/bridge/",
            type:"GET",
            headers:{
                "Access-Control-Allow-Origin":"*"
            }

        }).done((data)=>{

            if (data.ServerIP != "")
                $('[name=input-IP-bridge]').val(data.ServerIP)
                         
            if (data.Port != "")
                $('[name=input-port-bridge]').val(data.Port)                
            
        }).fail(()=>{
            Show_ErrorSweetToast("Error","Unable to upload info from server");       
        });

    });

    //btn save bridge's info
    $("[name=save-bridge]").on("click",function(){

        if (StateSimulator) {
            Show_ErrorSweetToast("Simulator in running", "Unable change data");
            return
        }
        
        var IPAddr = $("[name=input-IP-bridge]").val();
        var Port = $("[name=input-port-bridge]").val();

        //validation
        var ValidIP = IsValidIP(IPAddr);
        var ValidPort = Port < 65536 && Port > 0 ? true : false;

        var val = ValidIP && ValidPort;
        if(!val){
            ValidationInput($("[name=input-IP-bridge]"), ValidIP)
            ValidationInput($("[name=input-port-bridge]"), ValidPort)
            
            Show_ErrorSweetToast("Error", "Values are incorrect")

            return
        }

        //create file JSON
        var jsonData = JSON.stringify({
            "ServerIP" : IPAddr,
            "Port" : Port
        });

        //ajax
        $.post("http://localhost:8000/api/bridge/save", jsonData,"json")
        .done((data)=>{

            if (data.status == null)
                Show_SweetToast("Data saved","");  
           
        }).fail((data)=>{
            
            Show_ErrorSweetToast("Error", data.statusText); 

        });

    });

    //********************* Common *********************

    $("[name^=input-name-]").on("keyup",function(){
        $(this).removeClass("is-invalid is-valid")      
    });

    //ip address (also for real gw)
    $("[name^=input-IP]").on("blur keyup",function(){
        
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

        var address = $(this).parents("#modal-send-data").attr("data-addr");

        var ok = CanExecute();
        if (ok) //ok true: SIM off e DEV off
           Show_ErrorSweetToast("Unable send uplink","Simulator is stopped");
        else{

            var data ={
                "DevEUI": address,
                "MType": $(this).parents("#modal-send-data").attr("data-mtype"),
                "Payload": $(this).parents("#modal-send-data").find("[name=send-payload]").val()
            };
  
            socket.emit("send-uplink",data);
            
        }

        $('#modal-send-data').modal('toggle');

    });

    $("[name=periodicity]").on("blur keyup",function(){

        var value = $(this).val();
        var valid = IsValidNumber(value,-1, 8);     
        
        ValidationInput($(this),valid);

    });

    $("#submit-send-mac-command").on("click",function(){

        var address = $(this).parents("#modal-pingSlotInfoReq").attr("data-addr");

        var ok = CanExecute();
        if (ok)
            Show_ErrorSweetToast("Unable send MAC Command","Simulator is stopped");
        else{

            var valid = IsValidNumber($("[name=periodicity]").val(),-1, 8);
            if (valid){
                var data = {
                    "DevEUI": address,
                    "CID": "PingSlotInfoReq",
                    "Periodicity": Number($(this).parents("#modal-pingSlotInfoReq").find(" [name=periodicity]").val()),
                }
        
                socket.emit("send-MACCommand",data);
            }else
                return
           
        }        
        
        $('#modal-pingSlotInfoReq').modal('toggle');

    });

    $("#submit-new-location").on("click",function(){

        var address = $(this).parents("#modal-location").attr("data-addr");
        
        var ok = CanExecute();
        if (ok)
            Show_ErrorSweetToast("Unable change location","Simulator is stopped");
        else{

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
                "DevEUI": address,
                "Latitude":Number(latitude.val()),
                "Longitude":Number(longitude.val()),
                "Altitude":Number(altitude.val())
            }

            socket.emit("change-location",data,(response)=>{

                var index = Devices.findIndex((x) => x.Info.DevEUI == address);

                if (response){

                    Devices[index].Info.Location.Latitude = Number(latitude.val());
                    Devices[index].Info.Location.Longitude = Number(longitude.val());
                    Devices[index].Info.Location.Altitude = Number(altitude.val());

                    var latlng = L.latLng(Number(latitude.val()),Number(longitude.val()));

                    UpdateMarker(address, null, latlng, false);

                    Show_iziToast(Devices[index].Info.Name+" changed location","");
                }else
                    Show_iziToast(Devices[index].Info.Name+" may be turned off","");
                
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

            var data ={
                "DevEUI": address,
                "MType": mtype,
                "Payload": $("#payload-modal").val()
            };
    
            socket.emit("change-payload",data);
    
            for (var i=0; i < Devices.length ; i++){
                
                if (Devices[i].Info.DevEUI == data.DevEUI){
    
                    Devices[i].Info.Status.MType = data.MType;
                    Devices[i].Info.Status.Payload = data.Payload;
                    break;
                    
                }
    
            }
        }

        $('#modal-change-payload').modal('toggle');
            
    });

});

function Init(){
    
    //list of gateways
    $.ajax({
        url: "http://127.0.0.1:8000/api/gateways",
        type:"GET",
        headers:{
            "Access-Control-Allow-Origin":"*"
        }

    }).done((data)=>{

        Gateways = data

        data.forEach(element => {
            
            Add_ItemList($("#list-gateways"),element.Info.MACAddress, element.Info.Name);
   
            AddMarker(element.Info.MACAddress,element.Info.Name,
                L.latLng(element.Info.Location.Latitude, element.Info.Location.Longitude),
                true);           
                  
        });

        LoadListHome();

    }).fail((data)=>{
        console.log("fail:"+data)
    });

    //list of devices
    $.ajax({
        url: "http://127.0.0.1:8000/api/devices",
        type:"GET",
        headers:{
            "Access-Control-Allow-Origin":"*"
        }

    }).done((data)=>{

        Devices = data;
        
        data.forEach(element => {

            Add_ItemList($("#list-devices"),element.Info.DevEUI, element.Info.Name);
            
            AddMarker(element.Info.DevEUI, element.Info.Name,
                L.latLng(element.Info.Location.Latitude, element.Info.Location.Longitude),
                false);               

        });

        LoadListHome();

    }).fail((data)=>{
        console.log("fail:"+ data)
    });

}

//********************* Event *********************
function KeyUp_ValidAddress(element, event, bytes){

    var value = $(element).val().replaceAll(' ','');
    value = value.toLowerCase();

    if(event.keyCode!=8){

        if (value.length !=0 && event.which !=32){//insert :
            
            let valtmp = value.replaceAll(':','');

            if (valtmp.length % 2 == 0 && valtmp.length < bytes)                 
                $(element).val(value + ":");
            else
                $(element).val(value);
        }
    }
    
    return IsValidAddress($(element).val(),bytes == 16 ? true : false);

}

function Click_GenerateAddress(selector,pairs){

    var ok = SetData(selector,GenerateAddress(pairs));
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

function Add_ItemList(selector, mac, name){  

    var item = "<a href=\"#\" class=\"list-group-item list-group-item-action\" data-addr=\""+mac+"\">"+name+"</a>";
    selector.append(item);

}

function ShowList(selector, title){

    selector.addClass("active show");
    selector.siblings().removeClass("active show");
    $(".section-header h1").text(title);

}

function UpdateList(selector, obj, OldAddress, gw){

    selector.children("a[data-addr=\""+OldAddress+"\"]").text(obj.Name); 
    selector.children("a[data-addr=\""+OldAddress+"\"]").text(obj.Info.Name);
    if(gw)
        selector.children("a[data-addr=\""+OldAddress+"\"]").attr("data-addr",obj.Info.MACAddress);         
    else
        selector.children("a[data-addr=\""+OldAddress+"\"]").attr("data-addr",obj.Info.DevEUI);
    
        
}

function LoadListHome(){
    $("#list-home").empty();

    Devices.forEach(element =>{
        $("#list-home").append("<a href=\"#list-home\" class=\"text-blue list-group-item list-group-item-action\" data-addr=\""+element.Info.DevEUI+"\">"+element.Info.Name+"</a>");
    })

    Gateways.forEach(element =>{
        $("#list-home").append("<a href=\"#list-home\" class=\"text-orange list-group-item list-group-item-action\" data-addr=\""+element.Info.MACAddress+"\">"+element.Info.Name+"</a>");
    })
    
}
//********************* Validation ********************* 
function IsValidAddress(addr,eui64){

    var addrFormat, len;

    if (addr == "") return false;

    var value = addr.replaceAll(' ','');
    addr = value;

    if (eui64){//addr 64 bit
        addrFormat = /^(([a-f0-9]{2}[:]){7}[a-f0-9]{2}[,]?)/;
        len = value.length !=23 ? false:true;
    }else{ //addr 16 bit
        addrFormat = /^(([a-f0-9]{2}[:]){3}[a-f0-9]{2}[,]?)/;
        len = value.length !=11 ? false:true;
    }

    return addrFormat.test(value) && len;
    
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
    
    return value.match(ipFormat)  
        
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

function GenerateAddress(pairs){

    var hexDigits = "0123456789abcdef";
    var Address = "";

    for (var i = 0; i < pairs; i++) {
        Address += hexDigits.charAt(Math.round(Math.random() * 15));
        Address += hexDigits.charAt(Math.round(Math.random() * 15));
        if (i != pairs-1) Address += ":";
    }

    return Address;
}

function ChangeFormatValue(value, dot){

    if (dot){
        
        var parts = value.match(/.{1,2}/g);
        var new_value = parts.join(":");
    
        return new_value;
    }

    return value.replaceAll(":","");
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

    var tmp = GenerateAddress(PairsEUI64*2);//genera 16 valori separati da :

    return tmp.replaceAll(":","");
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

        for (var i = 0; i <= data.MaxRX1DROffset; i++)
            $("#dr-offset-rx1").append("<option value=\""+i+"\">"+i+"</option>");

        for (var i = 0; i < data.DataRate.length; i++){

            if (data.DataRate[i] != -1){

                DataRates.push(data.DataRate[i]);
                $("#datarate-uplink").append("<option value=\""+i+"\">"+i+"</option>");
                $("#datarate-rx-2").append("<option value=\""+i+"\">"+i+"</option>");

                var row = "<tr><th scope=\"row\">"+data.DataRate[i]+"</th>";
                row += "<td>"+data.Configuration[i]+"</td>";
                row += "<td>"+data.PayloadSize[i][0]+"</td>";
                row += "<td>"+data.PayloadSize[i][1]+"</tr>";
                
                $("#table-body").append(row);
            }
                
        }
        
        FrequencyRX2Default = data.FrequencyRX2;
        DatarateRX2Default = data.DataRateRX2;
        MinFrequency = data.MinFrequency;
        Maxfrequency = data.MaxFrequency;

        var table = "<a href=\"#\" class=\"show-table\" data-toggle=\"modal\" data-target=\"#modal-table\">(Show Table)</a>";
        
        $("#label-freq-rx2").text("Value in Hz. Default value is "+data.FrequencyRX2);  
        $("#label-datarate-rx2").html("Default value is "+data.DataRateRX2+". "+table);
        $("#label-datarate-uplink").html(table);

        //da gestire il payload

        if (loadDevice){
            $("#dr-offset-rx1").val(dev.Info.Configuration.RX1DROffset);
            $("#datarate-rx-2").val(dev.Info.RXs[1].DataRate);
            $("#datarate-uplink").val(dev.Info.Status.DataRate);
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

       MapGateway = new L.Map('map-gw').addLayer(osm).setView([CoordDefault, CoordDefault], 8);
       MapGateway.addControl(osmGeocoder);
     
       MapDevice = new L.Map('map-dev').addLayer(osmC).setView([CoordDefault,CoordDefault], 8);
       MapDevice.addControl(osmGeocoderDev);

       MapHome = new L.Map('map-home').addLayer(osmHome).setView([CoordDefault,CoordDefault], 1);
   
       MapModal = new L.Map('map-modal').addLayer(osmModal).setView([CoordDefault,CoordDefault], 8);
       MapModal.addControl(osmGeocoderModal);

       MarkerGateway = L.marker([CoordDefault, CoordDefault]).addTo(MapGateway);
       MarkerDevice = L.marker([CoordDefault, CoordDefault]).addTo(MapDevice);
       MarkerModal = L.marker([CoordDefault, CoordDefault]).addTo(MapModal);

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

function Change_Marker(e){
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

function ChangeView(latlng){

    switch (TurnMap){

        case 0:
            MapHome.setView(latlng, 8);
            break;
        case 1:
            MapDevice.setView(latlng, 8);
            break;
        case 2:
            MapGateway.setView(latlng, 8);
            break;
        case 3:
            MapModal.setView(latlng, 8);
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
    ChangeView([CoordDefault, CoordDefault]);
}

function AddMarker(Address, Name, latlng, isGw){
    
    var icon;
    var Marker;

    if(!isGw){

        icon = L.icon({
            iconUrl: './img/marker-icon.png',
            iconSize: [32, 41],
            iconAnchor:[19,41],
            popupAnchor:[1,-34],
            tooltipAnchor:[16,-28]
        });

        Marker = L.marker(latlng,{icon:icon});
        Marker.bindPopup(GetMenuDevicePopup(Address,Name)).on("popupopen",RegisterEventsPopup);
        Marker.on("click", Click_marker);
        Marker.on("popupclose", ClosePopup);
    }    
    else{

        icon = L.icon({
            iconUrl: './img/marker-yellow.png',
            iconSize: [32, 41],
            iconAnchor:[19,41],
            popupAnchor:[1,-34],
            tooltipAnchor:[16,-28]
        });

        Marker = L.marker(latlng,{icon:icon});
        Marker.bindPopup(GetMenuGatewayPopup(Address, Name,latlng)).on("popupopen",RegisterEventsPopupGw);    
    }
        
    MarkersHome.push({Address,Marker});
    Marker.addTo(MapHome);
  
}

function UpdateMarker(address, name, latlng, isGw){

    var index = MarkersHome.findIndex((x) => x.Address == address); 
    if (index != -1)
        MarkersHome[index].Marker.setLatLng(latlng);
    else
        AddMarker(address,name,latlng,isGw);
    
}

function ChangePositionMarker(address,latlng){

    switch (TurnMap){
        case 0:     
            var index = MarkersHome.findIndex((x) => x.Address == address); 
            if (index != -1)
                MarkersHome[index].Marker.setLatLng(latlng);

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

    var index = MarkersHome.findIndex((x) => x.Address == address);
    if(index != -1)
        MarkersHome[index].Marker.removeFrom(MapHome);

}

function GetMenuGatewayPopup(Address,Name, latlng){

    var menu = "<p class=\"text-center m-0 \">"+Name+"</p>";
        menu +="<p class=\"m-0 \">Latitude:"+latlng.lat+"</p>";
        menu +="<p class=\"m-0 \">Longitude:"+latlng.lng+"</p>";
        menu +="<div id=\"menu-actions\" data-addr=\""+Address+"\" class=\"mh-100 mt-1 overflow-auto list-group list-group-flush\">";
        menu += "<a href=\"#Turn\" class=\"list-group-item item-action p-2\" id=\"turn-gw\"> Toggle On/Off</a>";
        return menu;
}

function RegisterEventsPopupGw(){

    $("#turn-gw").on('click',function(){
        
        if (!StateSimulator) {
            Show_ErrorSweetToast("Simulator is stopped","");
            return
        }

        var event = "Turn-OFF-gw";
        var address = $(this).parent("#menu-actions").attr("data-addr");
        var index = GatewaysRunning.findIndex((x) => x.MACAddress == address);
        var active = GatewaysRunning[index].Active;
        
        if (!active)
            event = "Turn-ON-gw";

        socket.emit(event, address, (address, response)=>{

            var index = GatewaysRunning.findIndex((x) => x.MACAddress == address);

            if(response){
                GatewaysRunning[index].Active = !GatewaysRunning[index].Active;
                if (GatewaysRunning[index].Active)
                    Show_iziToast(Gateways[index].Info.Name +" Turn ON","");
                else
                    Show_iziToast(Gateways[index].Info.Name +" Turn OFF","");
            }                  
            
        });

        index = MarkersHome.findIndex((x) => x.Address == address);
        MarkersHome[index].Marker.closePopup();
       
    });
}

function GetMenuDevicePopup(Address, Name){

    var menu = "<p id=\"name-clicked-dev\" class=\"text-center m-0 \">"+Name+"</p>";
    menu +="<div id=\"menu-actions\" data-addr=\""+Address+"\" class=\"mh-100 mt-1 overflow-auto list-group list-group-flush\">";
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

        var event = "Turn-OFF-dev";
        var address = $(this).parent("#menu-actions").attr("data-addr");
        var index = DeviceRunning.findIndex((x) => x.DevEUI == address);
        var active = DeviceRunning[index].Active;

        if (!active)
            event = "Turn-ON-dev";

        socket.emit(event,address, (address, response)=>{
            
            var index = DeviceRunning.findIndex((x) => x.DevEUI == address);
            var i = Devices.findIndex((x) => x.Info.DevEUI == address);

            if(response){

                DeviceRunning[index].Active = !DeviceRunning[index].Active;  

                if (DeviceRunning[index].Active)
                    Show_iziToast(Devices[i].Info.Name +" Turn ON","");
                else
                    Show_iziToast(Devices[i].Info.Name +" Turn OFF","");         

            }
            else
                Show_iziToast(Devices[i].Info.Name, "Unable execute command");           
            
        });

        index = MarkersHome.findIndex((x) => x.Address == address);
        MarkersHome[index].Marker.closePopup();
       
    });

    $("#send-cdataUp").on('click',function(){

        var address = $(this).parent("#menu-actions").attr("data-addr");
        $('#modal-send-data').attr("data-addr",address);
        $('#modal-send-data').attr("data-mtype",ConfirmedData_uplink);
        $("#modal-send-data").find("[name=send-payload]").val("");

        var index = MarkersHome.findIndex((x) => x.Address == address);
        MarkersHome[index].Marker.closePopup();

    });

    $("#send-uncdataUp").on('click',function(){

        var address = $(this).parent("#menu-actions").attr("data-addr");
        $('#modal-send-data').attr("data-addr",address);
        $('#modal-send-data').attr("data-mtype",UnConfirmedData_uplink);
        $("#modal-send-data").find("[name=send-payload]").val("");

        var index = MarkersHome.findIndex((x) => x.Address == address);
        MarkersHome[index].Marker.closePopup();

    });

    $(".mac-command").on('click',function(){

        var address = $(this).parent("#menu-actions").attr("data-addr");
        var cmd = $(this).attr("data-cmd");

        if (cmd == "PingSlotInfoReq"){
            $("[name=periodicity]").val("");
            $('#modal-pingSlotInfoReq').attr("data-addr",address);
            return;
        }
        else{

            var ok = CanExecute();
            if (ok){
                Show_ErrorSweetToast("Unable send MAC Command","Simulator is stopped");
            }else{
                
                var data = {
                    "DevEUI": address,
                    "CID":cmd
                }
               
                socket.emit("send-MACCommand",data);
            }

        }

        var index = MarkersHome.findIndex((x) => x.Address == address);
        MarkersHome[index].Marker.closePopup();

    });

    $("#change-location").on("click",function(){

        var address = $(this).parent("#menu-actions").attr("data-addr");
        $('#modal-location').attr("data-addr",address);

        $("[name=input-latitude]").val("");
        $("[name=input-longitude]").val("");
        $("[name=input-altitude]").val("");

        var index = MarkersHome.findIndex((x) => x.Address == address);
        MarkersHome[index].Marker.closePopup();
        TurnMap = 3;

    });

    $("#change-payload").on("click",function(){

        var address = $(this).parent("#menu-actions").attr("data-addr");
        $('#modal-change-payload').attr("data-addr",address);

        $("#payload-modal").val("")

        var index = MarkersHome.findIndex((x) => x.Address == address);
        MarkersHome[index].Marker.closePopup();

    });
  
}

function DrawRange(latLng, range){
    Circle = new L.circle([latLng.lat, latLng.lng], {
        color: '#008AFF',
        fillColor: '#008AFF',
        fillOpacity: 0.5,
        radius: range
    }).addTo(MapHome);
}

function Click_marker(){
    
    var content = this._popup._content;
    var start = content.indexOf("data-addr=\"")
    var address = content.substring(start+11, start+27);
    //ottengo l'indirizzo dal popup del marker
    //non può essere recuperato dall'evento popupopen in quanto l'indirizzo non è aggiornato

    var index = Devices.findIndex((x) => x.Info.DevEUI == address);
    var range = Devices[index].Info.Configuration.Range;
    var lat = Devices[index].Info.Location.Latitude;
    var lng = Devices[index].Info.Location.Longitude;
    var latlng = new L.latLng(lat,lng);

    DrawRange(latlng, range);
}

function ClosePopup(){
    Circle.removeFrom(MapHome);
}

//********************* Gateway *********************
function CleanInputGateway(){

    $(".section-header h1").text("Add new Gateway");

    $("#add-gw input").val("");
    $("#add-gw input").removeClass("is-valid is-invalid");

    $("#choose-type label").removeClass("focus active");

    $("[id$=-gw]").prop("checked",false);
    $("#info-gw").addClass("hide");

    $("#div-buttons-gw").removeData("addr");

    CleanMap();

    ChangeStateInputGateway(false, null);

}

function Load_Gateway(gw) {

    var MAC = ChangeFormatValue(gw.Info.MACAddress, true);

    $("[name=input-name-gw]").val(gw.Info.Name);
    $("[name=input-MAC-gw]").val(MAC);
    $("#checkbox-active-gw").prop("checked",gw.Info.Active);

    if (!gw.Info.TypeGateway){
        $("#virtual-gw").parent("label").addClass("focus active");
        $("#info-virtual-gw").removeClass("hide");
        $("#info-real-gw").addClass("hide");
    }else{
        $("#real-gw").parent("label").addClass("focus active");
        $("#info-virtual-gw").addClass("hide");
        $("#info-real-gw").removeClass("hide");
    }
    
    $("#info-gw").removeClass("hide");

    //virtual
    $("[name=input-KeepAlive]").val(gw.Info.KeepAlive);

    //real
    $("[name=input-IP-gw]").val(gw.Info.Address);
    $("[name=input-port-gw]").val(gw.Info.Port);
    
    //map
    var latlng =  L.latLng(gw.Info.Location.Latitude, gw.Info.Location.Longitude); 
    ChangePositionMarker(-1,latlng);
    ChangeView(latlng);

    ChangeCoords(latlng);
    $("#add-gw [name=input-altitude]").val(gw.Info.Location.Altitude);
           
    $("#gws").removeClass("show active");
    $("#add-gw").addClass("show active");
    $(".section-header h1").text("Update Gateway");

    ChangeStateInputGateway(true, gw.Info.MACAddress);
    setTimeout(()=>{
        MapGateway.invalidateSize();
    },300);

}

function ChangeStateInputGateway(value,mac){

    $("#add-gw input").prop({"disabled":value,"readonly":value});
    $("#choose-type label").prop({"disabled":value,"readonly":value});
    
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

    var MACAddress = $("#div-buttons-gw").data("addr");
    
    swal({
        title: 'Are you sure?',
        text: 'Once deleted, you will not be able to recover this gateway!',
        icon: 'warning',
        buttons: true,
        dangerMode: true,
        })
        .then((willDelete) => {
        if (willDelete) {

            var jsonData = JSON.stringify({

                "Info":{
                    "MACAddress" : MACAddress
                }
 
            });

            //ajax
            $.post("http://127.0.0.1:8000/api/del-gateway",jsonData, "json")
            .done((data)=>{
            
                if (data.status){

                    $("#list-gateways").children("a[data-addr=\""+MACAddress+"\"]").remove();

                    var index = Gateways.findIndex((x) => x.Info.MACAddress == MACAddress);
                    if (index > -1) 
                        Gateways.splice(index, 1);

                    RemoveMarker(MACAddress);
                    ShowList($("#gws"),"List gateways");
                    Show_SweetToast('Gateway has been deleted!',"");

                }
                else
                    Show_ErrorSweetToast("Error","Gateway didn't deleted. It could be active");                                             

            }).fail((data)=>{
                Show_ErrorSweetToast("Error",data.statusText); 
            });            

        } 

        });
    
}

function Click_SaveGateway(){

    var MACAddress = $("#div-buttons-gw").data("addr");

    var Location, TypeGateway, valid;
    var index = -1;

    var NameGateway = $("[name=input-name-gw]");
    var MACGateway = $("[name=input-MAC-gw]");
    var IsActive = $("#checkbox-active-gw").prop("checked");  
    var KeepAlive = $("[name=input-KeepAlive]");
    var IPGateway = $("[name=input-IP-gw]");
    var PortGateway = $("[name=input-port-gw]");
    
    if ($("#virtual-gw").parent("label").hasClass("active")){//virtual

        TypeGateway = false;

        if(KeepAlive.val() == ""){

            valid = true;
            KeepAlive.val(KeepAliveDefault);

        } else
            valid = KeepAlive.val() <= 0 ? false : true; 

        ValidationInput(KeepAlive, valid);

    }else if ($("#real-gw").parent("label").hasClass("active")){//real

        TypeGateway = true;
        
        valid = IsValidIP(IPGateway.val());
        ValidationInput(IPGateway, valid);

        var validPort = IsValidNumber(PortGateway.val(),-1,65536);
        ValidationInput(IPGateway, validPort);

        valid = validPort ? valid : false;
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

    Location = {
        "Latitude":Number(latitude.val()),
        "Longitude": Number(longitude.val()),
        "Altitude": Number(altitude.val())
    }

    ValidationInput(latitude, validLat);
    ValidationInput(longitude, validLng);
    ValidationInput(altitude, true);

    if (MACAddress != undefined && MACAddress != "")// update gateway
        index = Gateways.findIndex((x) => x.Info.MACAddress == MACAddress);    

    //validation
    var validNameGateway = NameGateway.val() == "" ? false : true;  
    var validMACGateway = IsValidAddress(MACGateway.val(),true) ? true : false; 

    ValidationInput(NameGateway, validNameGateway);   
    ValidationInput(MACGateway, validMACGateway);

    if(!validMACGateway || !validNameGateway || !valid || 
        !validLat || !validLng ){//Error
         
        Show_ErrorSweetToast("Error: values are incorrect","");
        return;
    }

    var gw ={
        "Info":{
            "Active": IsActive,
            "Name" : NameGateway.val(),
            "MACAddress" : ChangeFormatValue(MACGateway.val(),false),//tolgo i :
            "KeepAlive": Number(KeepAlive.val()),
            "TypeGateway":TypeGateway,
            "Address":IPGateway.val(),
            "Port": PortGateway.val(),
            "Location":Location
        }
    };

    //file JSON
    var jsonData= JSON.stringify({
        "Gateway":gw,
        "Index":index        
    });

    if (MACAddress == undefined ||
        MACAddress == ""){//new gateway
        //ajax  
        $.post("http://localhost:8000/api/add-gateway",jsonData, "json")
        .done((data)=>{

            switch (data.code){
                case 0:
                
                    Gateways.push(gw);

                    AddMarker(gw.Info.MACAddress,gw.Info.Name,
                        L.latLng(gw.Info.Location.Latitude,gw.Info.Location.Longitude),
                        true);
 
                    
                    Add_ItemList($("#list-gateways"),gw.Info.MACAddress, gw.Info.Name);
                    CleanInputGateway();

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
            Show_ErrorSweetToast("Error",data.statusText);   
        });

    } else{//update gateway

        $.post("http://localhost:8000/api/up-gateway",jsonData, "json")
        .done((data)=>{

            switch (data.code){
                case 0:

                    Gateways[index] = gw;

                    var latlng = L.latLng(gw.Info.Location.Latitude, gw.Info.Location.Longitude);      
                    UpdateMarker(gw.Info.MACAddress,gw.Info.Name,latlng,true);
                
                    UpdateList($("#list-gateways"),gw, MACAddress, true);
                    ShowList($("#gws"),"List gateways");

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
            Show_ErrorSweetToast("Error",data.statusText);    
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

    $("#add-dev input").val("");
    $("#add-dev input").removeClass("is-valid is-invalid");
    $("#add-dev [type=checkbox]").prop("checked",false);

    $("#region").val(-1);

    $("select").removeClass("is-valid is-invalid");

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

    var valDevEUI = ChangeFormatValue(dev.Info.DevEUI,true);

    //general
    $("[name=checkbox-active-dev]").prop("checked",dev.Info.Status.Active);
    $("[name=input-name-dev]").val(dev.Info.Name);
    $("[name=input-devEUI]").val(valDevEUI);
    $("#region").val(dev.Info.Configuration.Region);
    
    SetParameters(dev.Info.Configuration.Region, true,dev);

    //activation
    $("#checkbox-otaa-dev").prop("checked",dev.Info.Configuration.SupportedOtaa);
    if(dev.Info.Configuration.SupportedOtaa)
        $("[name=input-key-app]").val(dev.Info.AppKey);    
    else{
        $("[name=input-devAddr]").val(ChangeFormatValue(dev.Info.DevAddr,true));
        $("[name=input-key-nwkS]").val(dev.Info.NwkSKey);
        $("[name=input-key-appS").val(dev.Info.AppSKey);
    }

    //class A
    $("[name=input-rx-1-delay]").val(dev.Info.RXs[0].Delay);
    $("[name=input-rx-1-duration]").val(dev.Info.RXs[0].DurationOpen);   
    $("[name=input-rx-2-delay]").val(dev.Info.RXs[1].Delay);
    $("[name=input-rx-2-duration]").val(dev.Info.RXs[1].DurationOpen);
    $("[name=input-frequency-rx-2]").val(dev.Info.RXs[1].Channel.FreqDownlink);   
    $("[name=input-ackTimeout]").val(dev.Info.Configuration.AckTimeout);
    $("#dr-offset-rx1").val(dev.Info.Configuration.RX1DROffset);

    //ClassB,C
    $("#classB-dev").prop("checked", dev.Info.Configuration.SupportedClassB);
    $("#classC-dev").prop("checked", dev.Info.Configuration.SupportedClassC);

    //frame settings
    
    $("[name=input-fport]").val(dev.Info.Status.InfoUplink.FPort);
    $("[name=input-retransmission]").val(dev.Info.Configuration.NbRetransmission)
    $("[name=input-fcnt]").val(dev.Info.Status.InfoUplink.FCnt);
    $("#datarate-uplink").val(dev.Info.Status.DataRate);

    $("[name=input-validate-counter]").prop("checked",dev.Info.Configuration.DisableFCntDown);  
    if (!dev.Info.Configuration.DisableFCntDown)
        $("[name=input-fcnt-downlink]").val(dev.Info.Status.FCntDown);

    //features
    $("[name=input-ADR]").prop("checked", dev.Info.Configuration.SupportedADR);
    $("[name=input-range]").val(dev.Info.Configuration.Range);
    
    //location
    var latlng =  L.latLng(dev.Info.Location.Latitude, dev.Info.Location.Longitude); 
    ChangePositionMarker(-1,latlng);
    ChangeView(latlng);

    ChangeCoords(latlng);
    $("#add-dev [name=input-altitude]").val(dev.Info.Location.Altitude);
    

    //payload
    $("[name=input-sendInterval]").val(dev.Info.Configuration.SendInterval);

    if(dev.Info.Status.MType == ConfirmedData_uplink){
        $("#confirmed").prop("checked",true);
        $("#unconfirmed").prop("checked",false);
    }else{
        $("#confirmed").prop("checked",false);
        $("#unconfirmed").prop("checked",true);
    }

    $("#fragments").prop("checked",dev.Info.Configuration.SupportedFragment);
    $("#truncates").prop("checked",!dev.Info.Configuration.SupportedFragment);
  
    $("#textarea-payload").val(dev.Info.Status.Payload);

    ChangeStateInputDevice(true,dev.Info.DevEUI);


    $("#devs").removeClass("show active");
    $("#add-dev").addClass("show active");
    $(".section-header h1").text("Update Device");

}

function Click_DeleteDevice(){

    var devEUI = $("#div-buttons-dev").data("addr");

    swal({
        title: 'Are you sure?',
        text: 'Once deleted, you will not be able to recover this device!',
        icon: 'warning',
        buttons: true,
        dangerMode: true,
        })
        .then((willDelete) => {
        if (willDelete) {
    
            var jsonData = JSON.stringify({
                "DevEUI" : devEUI
            });

            //ajax
            $.post("http://127.0.0.1:8000/api/del-device",jsonData, "json")
            .done((data)=>{
        
                if (data.status){
        
                    $("[id^=list-devices]").children("a[data-addr=\""+devEUI+"\"]").remove();

                    var index = Devices.findIndex((x) => x.Info.DevEUI == devEUI);
                    if (index > -1) {
                        Devices.splice(index, 1);
                    }

                    RemoveMarker(devEUI);
                    ShowList($("#devs"),"List devices");
                    Show_SweetToast("Device Deleted","");

                }
                else
                    Show_ErrorSweetToast("Error","Device didn't deleted. It could be active");

            }).fail((data)=>{
                Show_ErrorSweetToast("Error",data.statusText); 
            });
        }
    });   

}

function Click_SaveDevice(){

    var index = -1;
    var validation;

    //******************* general ***************************

    var active = $("[name=checkbox-active-dev]").prop("checked");
    var name = $("[name=input-name-dev]");
    var devEUI = $("[name=input-devEUI]");
    var region = $("#region");

    var validNameDevice = name.val() == "" ? false : true;  
    var validDevEUI = IsValidAddress(devEUI.val(),true);
    var validRegion = region.val() == -1 ? false: true;
    ValidationInput(name, validNameDevice);
    ValidationInput(devEUI, validDevEUI);
    ValidationInput(region, validRegion);

    validation = validDevEUI && validNameDevice && validRegion;

    //******************* activation ***************************

    var supportedOtaa = $("#checkbox-otaa-dev").prop("checked");
    var devAddr = $("[name=input-devAddr]");
    var NwkSKey = $("[name=input-key-nwkS]");
    var AppSKey = $("[name=input-key-appS");
    var AppKey = $("[name=input-key-app]");

    var valueDevAddr = "";
    var valueNwkSKey = "";
    var valueAppSKey = "";

    if (supportedOtaa){

        ValidationInput(AppKey, IsValidKey(AppKey.val()));
        validation = validation && IsValidKey(AppKey.val());

    }else{

        var validDevAddr = IsValidAddress(devAddr.val(), false);       
        var validNwkSkey = IsValidKey(NwkSKey.val());        
        var validAppSKey = IsValidKey(AppSKey.val());

        valueDevAddr = devAddr.val();
        valueNwkSKey = NwkSKey.val();
        valueAppSKey = AppSKey.val(); 

        ValidationInput(devAddr, validDevAddr);
        ValidationInput(NwkSKey, validNwkSkey);
        ValidationInput(AppSKey, validAppSKey);

        validation = validation && validDevAddr && validNwkSkey && validAppSKey;
    }
    
    //******************* class A ***************************

    var delayRX1 = $("[name=input-rx-1-delay]");
    var durationRX1 = $("[name=input-rx-1-duration]");
    var DROffsetRX1 = $("#dr-offset-rx1");
    var delayRX2 = $("[name=input-rx-2-delay]");
    var durationRX2 = $("[name=input-rx-2-duration]");
    var frequencyRX2 = $("[name=input-frequency-rx-2]");
    var datarateRX2 = $("#datarate-rx-2");
    var ackTimeout = $("[name=input-ackTimeout]");

    delayRX1.val(delayRX1.val() == "" ? DelayDefault : delayRX1.val());
    var validDelayRX1 = IsValidNumber(delayRX1.val(),0,Infinity);

    durationRX1.val(durationRX1.val() == "" ? DurationDefault : durationRX1.val());
    var validDurationRX1 = IsValidNumber(durationRX1.val(),0,Infinity);
    
    delayRX2.val(delayRX2.val() == "" ? DelayDefault : delayRX2.val());
    var validDelayRX2 = IsValidNumber(delayRX2.val(),0,Infinity);

    durationRX2.val(durationRX2.val() == "" ? DurationDefault : durationRX2.val());
    var validDurationRX2 = IsValidNumber(durationRX2.val(),0,Infinity);
 
    frequencyRX2.val(frequencyRX2.val() == "" ? FrequencyRX2Default : frequencyRX2.val());
    var validFrequencyRX2 = IsValidNumber(frequencyRX2.val(),MinFrequency-0.01,Maxfrequency+0.01);

    datarateRX2.val(datarateRX2.val() == -1 ? DatarateRX2Default : datarateRX2.val());
    ValidationInput(datarateRX2,true);

    ackTimeout.val(ackTimeout.val() == "" ? ACKTimeoutDefault : ackTimeout.val());
    var validAckTimeout= IsValidNumber(ackTimeout.val(),-1,4);

    ValidationInput(delayRX1,validDelayRX1);
    ValidationInput(durationRX1,validDurationRX1);
    ValidationInput(delayRX2,validDelayRX2);
    ValidationInput(durationRX2,validDurationRX2);
    ValidationInput(frequencyRX2,validFrequencyRX2);
    ValidationInput(ackTimeout,validAckTimeout);

    validation = validation && validDelayRX1 && validDelayRX2 && validDurationRX1;
    validation = validation && validDurationRX2 && validFrequencyRX2 && validAckTimeout;

    //******************* Class B e C ***************************

    var isClassBActive = $("#classB-dev").prop("checked");
    var isClassCActive = $("#classC-dev").prop("checked");

    //******************* frame's settings ***************************

    var datarate = $("#datarate-uplink");
    var fport = $("[name=input-fport]");
    var retransmission = $("[name=input-retransmission]")
    var Fcnt = $("[name=input-fcnt]");
    var disableFCntDown = $("[name=input-validate-counter]").prop("checked");
    var FCntDown = $("[name=input-fcnt-downlink]");

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
    if(!disableFCntDown){
    
        if(FCntDown.val() == "")
            validFcntDown = false;
        else
            validFcntDown = IsValidNumber(FCntDown.val(),-1,MaxValueCounter+1);
    } 
    
    ValidationInput(fport,validFport);
    ValidationInput(retransmission,validReply);
    ValidationInput(Fcnt,validFcnt);
    ValidationInput(FCntDown,validFcntDown);

    validation = validation && validFport && validReply && validFcnt && validFcntDown;

    //******************* features ***************************

    var SupportedADR = $("[name=input-ADR]").prop("checked");
    var range = $("[name=input-range]");

    $("[name=input-range]").val($("[name=input-range]").val() == "" ? RangeDefault : $("[name=input-range]").val())
    var validRange = IsValidNumber(range.val(),0,Infinity);

    ValidationInput(range,validRange);

    validation = validation && validRange;

    //******************* location ***************************

    var selector = GetMap();
    var latitude = selector.find("#coords").find(" [name=input-latitude]");
    var longitude = selector.find("#coords").find(" [name=input-longitude]");
    var altitude = selector.find("#coords").find(" [name=input-altitude]");

    var validLat = IsValidNumber(Number(latitude.val()),-90.01,90.01);
    var validLng = IsValidNumber(Number(longitude.val()),-180.01,180.01);

    var Location = {
        "Latitude":Number(latitude.val()),
        "Longitude": Number(longitude.val()),
        "Altitude":Number(altitude.val())
    }

    ValidationInput(latitude, validLat);
    ValidationInput(longitude, validLng);
    ValidationInput(altitude, true);

    validation = validation && validLat && validLng;
 
    //******************* payload ***************************

    var MType = $("#confirmed").prop("checked") ? ConfirmedData_uplink : UnConfirmedData_uplink; //true Confirmed
    var upInterval = $("[name=input-sendInterval]");
    var payload = $("#textarea-payload").val();

    upInterval.val(upInterval.val() == "" ? UplinkIntervalDefault : upInterval.val());
    var validInterval = IsValidNumber(upInterval.val(),-1,Infinity);

    ValidationInput(upInterval,validInterval);

    validation = validation && validInterval;
    
    if (!validation){
        Show_ErrorSweetToast("Error","Values are incorrect");
        return;
    }

    var id = $("#div-buttons-dev").data("addr");
    if (id != undefined && id != "")// update device       
        index = Devices.findIndex((x) => x.Info.DevEUI == id);


    //JSON creation
    var dev ={
    "Info":{
            "Name" : name.val(),
            "DevEUI" : ChangeFormatValue(devEUI.val(),false),
            "AppKey": AppKey.val(),
            "DevAddr": ChangeFormatValue(valueDevAddr,false),
            "NwkSKey": valueNwkSKey,
            "AppSKey": valueAppSKey,
            "Location":Location,
            "Status":{
                "Active": active,
                "DataRate": Number(datarate.val()),
                "InfoUplink":{
                    "FPort": Number(fport.val()),
                    "FCnt": Number(Fcnt.val()),
                },
                "MType": MType,
                "Payload":payload,
                "FCntDown": Number(FCntDown.val())
            },
            "Configuration":{
                "Region":Number(region.val()),
                "AckTimeout":Number(ackTimeout.val()),
                "RX1DROffset":Number(DROffsetRX1.val()),
                "SupportedADR":SupportedADR,
                "SupportedOtaa":supportedOtaa,
                "SupportedFragment":$("#fragments").prop("checked"),
                "SupportedClassB":isClassBActive,
                "SupportedClassC":isClassCActive,
                "Range":Number(range.val()),
                "DisableFCntDown":disableFCntDown,
                "SendInterval":Number(upInterval.val()),
                "NbRetransmission":Number(retransmission.val()),
            },
            "RXs":[
                {
                    "Delay":Number(delayRX1.val()),
                    "DurationOpen":Number(durationRX1.val())
                },{
                    "Channel":{
                        "Active":true,
                        "FreqDownlink": Number(frequencyRX2.val())
                    },
                    "Delay":Number(delayRX2.val()),
                    "DurationOpen":Number(durationRX2.val()),
                    "DataRate":Number(datarateRX2.val())
                }
            ]
        }    
    };
    
    var jsonData = JSON.stringify({
        "Device":dev,
        "Index":index     
    });
    
    if (id == undefined ||
        id == ""){//new device
        //ajax  

        $.post("http://localhost:8000/api/add-device",jsonData, "json")
        .done((data)=>{

            switch (data.code){
                case 0: //OK
                    
                    AddMarker(dev.Info.DevEUI,dev.Info.Name,
                        L.latLng(dev.Info.Location.Latitude, dev.Info.Location.Longitude),
                        false);
    
                    Devices.push(dev);
    
                    Add_ItemList($("#list-devices"),dev.Info.DevEUI, dev.Info.Name);//ui
                    ShowList($("#devs"),"List devices" );            

                    CleanInputDevice();

                    Show_SweetToast("New Device saved","");
                            
                    return;

                case 1:// same name

                    NameDevice.addClass("is-invalid");
                    DevEUI.siblings(".invalid-feedback").text(data.status);

                    return;

                case 2:

                    NameDevice.addClass("is-invalid");
                    DevEUI.addClass("is-invalid");
                    DevEUI.siblings(".invalid-feedback").text(data.status);

                    return;
                   
            }
                 
        }).fail((data)=>{  
            Show_ErrorSweetToast("Error", data.statusText);   
        });

    } else{//update device

        $.post("http://localhost:8000/api/up-device",jsonData, "json")
        .done((data)=>{

            switch (data.code){
                case 0:

                    Devices[index] = dev;            

                    var latlng = L.latLng(dev.Info.Location.Latitude, dev.Info.Location.Longitude);
                    UpdateMarker(dev.Info.DevEUI,dev.Info.Name,latlng,false);
                    
                    UpdateList($("#list-devices"),dev, id, false);
                    ShowList($("#devs"),"List devices");

                    Show_SweetToast("Device updated","");

                    CleanInputDevice();

                    return;

                case 1:// same name

                    name.addClass("is-invalid");
                    break;

                case 2:

                    devEUI.addClass("is-invalid");
                    break;

                case 3:
                    Show_ErrorSweetToast("Error",data.status); 
                    return;

            }   
            Show_ErrorSweetToast("Error",data.status);
            
        }).fail((data)=>{    
            Show_ErrorSweetToast("Error",data.statusText);   
        });
    }
    
}

//********************* Common *********************
function Click_Edit(element, FlagGw){
 
    var addr = $("#div-buttons-dev").data("addr");

    if(FlagGw){
        addr = $("#div-buttons-gw").data("addr");   
    }

    var ok = CanExecute();
    if (FlagGw && !ok){
        Show_ErrorSweetToast("Error","Unable edit gateway because Simulator is running and gateway is active");
        return
    }

    $(element).hide();   
    
    if (FlagGw) //Edit gateway
        ChangeStateInputGateway(false,null);        
    else
        ChangeStateInputDevice(false,null);

    $(element).siblings("button").show();
   
}
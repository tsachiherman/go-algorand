<?php $this->layout('app:layout');?>

<div class="container" style="width:81%">
    <div class="alert alert-success" style="text-align:center;font-size:30px" role="alert">
    Vote Route</div>
</div>


<div style="width:100%; text-align:center;overflow-x: scroll;">
<div id="curve_chart" style="width: 95%; display:inline-block; "></div>
<div id="sankey_chart" style="width: 95%; height: 100%; display:none;overflow-x: auto;"></div>
</div>
<div id="map_div" style="width:100%; height: 100%; display:none;"></div>

<?
        if ($graphstyle == "0") {
            ?>
<style>
 table.google-visualization-orgchart-table {
   border-collapse: separate !important;
 }

 table.graphtableclass1 {
    width:200px;
    text-align:left;
    overflow-x: auto;
    display: block;
 }
 table.graphtableclass2 {
    width:200px;
    text-align:left;
    overflow-x: auto;
    display: block;
 }
 table.graphtableclass3 {
    width:200px;
    text-align:left;
    overflow-x: auto;
    display: block;
 }
 table.graphtableclass4 {
    width:200px;
    text-align:left;
    overflow-x: auto;
    display: block;
 }
 table.graphtableclass5 {
    width:200px;
    text-align:left;
    overflow-x: auto;
    display: block;
 }
 table.graphtableclass6 {
    width:200px;
    text-align:left;
    overflow-x: auto;
    display: block;
 }
 table.graphtableclass7 {
    width:200px;
    text-align:left;
    overflow-x: auto;
    display: block;
 }
 table.graphtableclass8 {
    width:200px;
    text-align:left;
    overflow-x: auto;
    display: block;
 }


 td.google-visualization-orgchart-nodesel {
    border: 2px solid #e3ca4b;
    background-color: #fff7ae;
    background: -webkit-gradient(linear, left top, left bottom, from(#fff7ae), to(#eee79e));
    background-color: rgba(0, 0, 0, 0);
    background-position-x: 0%;
    background-position-y: 0%;
    background-repeat: repeat;
    background-attachment: scroll;
    background-image: linear-gradient(rgb(255, 247, 174) 0%, rgb(238, 231, 158) 100%);
    background-size: auto;
    background-origin: padding-box;
    background-clip: border-box;
}
</style>

<script type="text/javascript">
<?
            echo "document.getElementById('curve_chart').style.display = \"inline-block\";\r\n";
            echo "google.charts.load('current', {'packages':['orgchart']});\r\n";
            echo "google.charts.setOnLoadCallback(drawChart);\r\n";
      ?>
      function drawChart() {

        var data = new google.visualization.DataTable();
        data.addColumn('string', 'Name'); // content; v = self key, f = display
        data.addColumn('string', 'Manager'); // parent id
        data.addColumn('string', 'ToolTip'); // tooltip

        <?php
        $connectionsArray = [];
        foreach($connections as $conn) {
            array_push($connectionsArray, $conn);
        }

        $rowsCount = 1;
        $rowLevel = 1;
        $seenVoteRelays = [0];
        foreach($voterConnections as $conn) {
            echo "data.addRows([[{'v':'" . $conn->guid . "', 'f':'<table class=graphtableclass" . $rowLevel ."><tr><td>Node GUID</td><td>" . $conn->name . "</td></tr><tr><td>Time</td><td>" . $firstVoteTimestamp . "</td></tr></table>'}, '', 'Vote Origin']]);\r\n";
            break;
        }
        $pendingConn = [];
        foreach($voterConnections as $conn) {
            $otherrelay = '';
            $relay = '';
            if ($conn->otherguid == '') {
                continue;
            }
            foreach($connections as $conn2) {
                if ($conn2->otherguid == $conn->otherguid) {
                    $otherrelay = $conn2->otherrelay;
                    break;
                }
                if ($conn2->guid == $conn->otherguid) {
                    $otherrelay = $conn2->relay;
                    break;
                }
                if ($conn2->otherguid == $conn->guid) {
                    $relay = $conn2->otherrelay;
                    break;
                }
                if ($conn2->guid == $conn->guid) {
                    $relay = $conn2->relay;
                    break;
                }
            }
            $o = (object) [
                'otherguid' => $conn->otherguid,
                'othername' => $conn->othername,
                'guid' => $conn->guid,
                'otherrelay' => $otherrelay,
                'relay' => $relay
            ];
            array_push($pendingConn, $o);
        }
        while (count($pendingConn) > 0) {
            $iterationItemCount = count($pendingConn);
            $rowLevel++;
            while ($iterationItemCount > 0) {
                $iterationItemCount--;
                $conn = array_shift($pendingConn);
                // add the item for the destination to our graph.
                if ($conn->otherrelay != "") {
                    echo "        data.addRows([[{'v':'" . $conn->otherguid . "', 'f':'<table class=graphtableclass" . $rowLevel ."><tr><td>Node GUID</td><td>" . $conn->othername . "</td></tr><tr><td>Relay Name</td><td>". $conn->otherrelay."</td></tr></table>'}, '" . $conn->guid . "', 'Vote Flow']]);\r\n";
                } else {
                    echo "        data.addRows([[{'v':'" . $conn->otherguid . "', 'f':'<table class=graphtableclass" . $rowLevel ."><tr><td>Node GUID</td><td>" . $conn->othername . "</td></tr></table>'}, '" . $conn->guid . "', 'Vote Flow']]);\r\n";
                }
                // see if we have this relay in our `seenAuthRelays` list.
                foreach($seenAuthRelays as $seenRelay) {
                    if ($seenRelay->relay == $conn->otherrelay) {
                        array_push($seenVoteRelays, $rowsCount);
                        break;
                    }
                }
                $rowsCount++;
                // find all the links from otherguid to other relays and add them to the pending connections array.
                for ($i = count($connectionsArray)-1; $i >= 0; $i--){
                    if ($conn->otherguid == $connectionsArray[$i]->guid) {
                        array_push($pendingConn, $connectionsArray[$i]);
                        unset($connectionsArray[$i]);
                        $connectionsArray = array_values($connectionsArray);
                    } else if ($conn->otherguid == $connectionsArray[$i]->otherguid) {
                        unset($connectionsArray[$i]);
                        $connectionsArray = array_values($connectionsArray);
                    }
                }

            }
        };

        foreach($seenVoteRelays as $seenVoteRelay) {
            echo "data.setRowProperty(" . $seenVoteRelay . ", 'style', 'background-image: linear-gradient(rgb(247, 255, 174) 0%, rgb(231, 238, 158) 100%); border-color: #cae34b;'); \r\n";
            echo "data.setRowProperty(" . $seenVoteRelay . ", 'selectedStyle', 'background-image: linear-gradient(rgb(247, 255, 174) 0%, rgb(231, 238, 158) 100%); border-color: #e3ca4b;'); \r\n";
        };
        ?>


        // Create the chart.
        var chart = new google.visualization.OrgChart(document.getElementById('curve_chart'));
        // Draw the chart, setting the allowHtml option to true for the tooltips.
        chart.draw(data, {
            'allowHtml':true,
            'allowCollapse':true,
            'size': 'small'            
            });
      }
      </script>
      <?
        }
      ?>
    

    <?
      if ($graphstyle == "1") {      
      ?>
<script type="text/javascript">
      <?
        echo "document.getElementById('sankey_chart').style.display = \"inline-block\";\r\n";
        echo "google.charts.load('current', {'packages':['sankey']});\r\n";
        echo "google.charts.setOnLoadCallback(drawSankeyChart);\r\n";
      
      ?>

      function drawSankeyChart() {

        var data = new google.visualization.DataTable();
        data.addColumn('string', 'From');
        data.addColumn('string', 'To');
        data.addColumn('number', 'Weight');
        data.addColumn({type:'string', role:'style'});


        <?php
        $connectionsArray = [];
        foreach($connections as $conn) {
            array_push($connectionsArray, $conn);
        }

        $rowsCount = 1;
        $rowLevel = 1;
        $seenVoteRelays = [0];
        /*foreach($voterConnections as $conn) {
            echo "data.addRows([[{'v':'" . $conn->guid . "', 'f':'<table class=graphtableclass" . $rowLevel ."><tr><td>Node GUID</td><td>" . $conn->name . "</td></tr><tr><td>Time</td><td>" . $firstVoteTimestamp . "</td></tr></table>'}, '', 'Vote Origin']]);\r\n";
            break;
        }*/
        $pendingConn = [];
        foreach($voterConnections as $conn) {
            $otherrelay = '';
            $relay = $conn->name;
            if ($conn->otherguid == '') {
                continue;
            }
            foreach($connections as $conn2) {
                if ($conn2->otherguid == $conn->otherguid) {
                    $otherrelay = $conn2->otherrelay;
                    break;
                }
                if ($conn2->guid == $conn->otherguid) {
                    $otherrelay = $conn2->relay;
                    break;
                }
                if ($conn2->otherguid == $conn->guid) {
                    $relay = $conn2->otherrelay;
                    break;
                }
                if ($conn2->guid == $conn->guid) {
                    $relay = $conn2->relay;
                    break;
                }
            }
            $o = (object) [
                'otherguid' => $conn->otherguid,
                'othername' => $conn->othername,
                'guid' => $conn->guid,
                'otherrelay' => $otherrelay,
                'relay' => $relay
            ];
            array_push($pendingConn, $o);
        }
        while (count($pendingConn) > 0) {
            $iterationItemCount = count($pendingConn);
            $rowLevel++;
            while ($iterationItemCount > 0) {
                $iterationItemCount--;

                $conn = array_shift($pendingConn);
                // add the item for the destination to our graph.
                echo "data.addRows([['" . $conn->guid . "', '" . $conn->otherguid . "', 1, '']]);\r\n";
                //echo "data.addRows([['" . $conn->relay . "', '" . $conn->otherrelay . "', 1, '']]);\r\n";
                echo " // " . $conn->relay . " ==> " . $conn->otherrelay . "\r\n";
                
                // see if we have this relay in our `seenAuthRelays` list.
                foreach($seenAuthRelays as $seenRelay) {
                    if ($seenRelay->relay == $conn->relay) {
                        array_push($seenVoteRelays, $rowsCount);
                        break;
                    }
                }
                $rowsCount++;
            
                // find all the links from otherguid to other relays and add them to the pending connections array.
                for ($i = count($connectionsArray)-1; $i >= 0; $i--){
                    if ($conn->otherguid == $connectionsArray[$i]->guid) {
                        array_push($pendingConn, $connectionsArray[$i]);
                        unset($connectionsArray[$i]);
                        $connectionsArray = array_values($connectionsArray);
                    } else if ($conn->otherguid == $connectionsArray[$i]->otherguid) {
                        unset($connectionsArray[$i]);
                        $connectionsArray = array_values($connectionsArray);
                    }
                }

            }
        };

        foreach($seenVoteRelays as $seenVoteRelay) {
            //echo "data.setRowProperty(" . $seenVoteRelay . ", 'style', 'background-image: linear-gradient(rgb(247, 255, 174) 0%, rgb(231, 238, 158) 100%); border-color: #cae34b;'); \r\n";
            //echo "data.setRowProperty(" . $seenVoteRelay . ", 'selectedStyle', 'background-image: linear-gradient(rgb(247, 255, 174) 0%, rgb(231, 238, 158) 100%); border-color: #e3ca4b;'); \r\n";
        };
        ?>


        // Create the chart.
        var chart = new google.visualization.Sankey(document.getElementById('sankey_chart'));
        // Draw the chart, setting the allowHtml option to true for the tooltips.
        chart.draw(data, {
            'height':1024,
            sankey: {
                link: {
                    color: { 
                        stroke: 'black', 
                        strokeWidth: 1,
                        colors: [ '#a6cee3', '#b2df8a', '#fb9a99', '#fdbf6f', '#cab2d6', '#ffff99', '#1f78b4', '#33a02c'],
                    } 
                },
                node: { 
                    colors: [ '#a6cee3', '#b2df8a', '#fb9a99', '#fdbf6f', '#cab2d6', '#ffff99', '#1f78b4', '#33a02c'],
                    nodePadding: 20,
                },
                iterations: 128,
            }
            });
      }
      </script>
      <?php
}
?>
<?
        if ($graphstyle == "2") {
?>
<style>
    html, body {
        height: 100%;
        margin: 0;
        padding: 0;
      }
</style>
<script type="text/javascript">
    var polylines = [];
    document.getElementById('map_div').style.display = "block";
    document.getElementById('map_div').style.height = "75%";
    var mapmap;

    function initMap() {
        mapmap = new google.maps.Map(document.getElementById('map_div'), {
          center: {lat: <?=$firstVote->long?>, lng: <?=$firstVote->lat?>},
          zoom: 3
        });

        drawMap();
    }    


    function drawMap() {
        <?php
        $markers = array();
        array_push($markers, 
                    (object) [
                        "long" => $firstVote->long,
                        "lat" => $firstVote->lat,
                        "hop" => 0,
                        "name" => $firstVote->sendertelemetryid ]
                );

        $connectionsArray = [];
        $polylines = array();
        foreach($connections as $conn) {
            array_push($connectionsArray, $conn);
        }

        $rowsCount = 1;
        $rowLevel = 1;
        $seenVoteRelays = [0];
        $seenVoteLong = array();
        array_push($seenVoteLong, "", $firstVote->long);

        foreach($voterConnections as $conn) {
 \           array_push($seenVoteLong, $conn->otherlong);
            break;
        }
        
        
        $pendingConn = [];
        foreach($voterConnections as $conn) {
            $otherrelay = '';
            $relay = '';
            if ($conn->otherguid == '') {
                continue;
            }
            if (in_array($conn->otherlong, $seenVoteLong)==true) {
                continue;
            }
            foreach($connections as $conn2) {
                if ($conn2->otherguid == $conn->otherguid) {
                    $otherrelay = $conn2->otherrelay;
                    break;
                }
                if ($conn2->guid == $conn->otherguid) {
                    $otherrelay = $conn2->relay;
                    break;
                }
                if ($conn2->otherguid == $conn->guid) {
                    $relay = $conn2->otherrelay;
                    break;
                }
                if ($conn2->guid == $conn->guid) {
                    $relay = $conn2->relay;
                    break;
                }
            }
            $o = (object) [
                'otherguid' => $conn->otherguid,
                'othername' => $conn->othername,
                'guid' => $conn->guid,
                'otherrelay' => $otherrelay,
                'relay' => $relay,
                'otherlong' => $conn->otherlong,
                'otherlat' => $conn->otherlat,
                'long' => $firstVote->long,
                'lat' => $firstVote->lat,
            ];
            array_push($pendingConn, $o);
            array_push($seenVoteLong, $conn->otherlong);
        }
        while (count($pendingConn) > 0) {
            $iterationItemCount = count($pendingConn);
            $rowLevel++;
            while ($iterationItemCount > 0) {
                $iterationItemCount--;
                $conn = array_shift($pendingConn);
                // add the item for the destination to our graph.
                if ($conn->otherrelay != "") {
                    array_push($markers, 
                        (object) [
                            "long" => $conn->otherlong,
                            "lat" => $conn->otherlat,
                            "hop" => $rowLevel,
                            "name" => $conn->othername ]
                    );
        
                } else {
                }
                array_push($polylines, 
                    (object) [
                        "long" => $conn->long,
                        "lat" => $conn->lat,
                        "otherlong" => $conn->otherlong,
                        "otherlat" => $conn->otherlat ]
                );

                // see if we have this relay in our `seenAuthRelays` list.
                foreach($seenAuthRelays as $seenRelay) {
                    if ($seenRelay->relay == $conn->otherrelay) {
                        array_push($seenVoteRelays, $rowsCount);
                        break;
                    }
                }
                $rowsCount++;
                // find all the links from otherguid to other relays and add them to the pending connections array.
                for ($i = count($connectionsArray)-1; $i >= 0; $i--){
                    if (array_search($connectionsArray[$i]->otherlong, $seenVoteLong)!=FALSE || $connectionsArray[$i]->otherlong=="") {
                        continue;
                    }
                    if ($connectionsArray[$i]->otherlong > 90 || $connectionsArray[$i]->otherlong < -90 ) {
                        continue;
                    }
                    if ($conn->otherguid == $connectionsArray[$i]->guid) {
                        array_push($seenVoteLong, $connectionsArray[$i]->otherlong);
                        array_push($pendingConn, (object) [
                            'otherguid' => $connectionsArray[$i]->otherguid,
                            'othername' => $connectionsArray[$i]->othername,
                            'guid' => $connectionsArray[$i]->guid,
                            'otherrelay' => $connectionsArray[$i]->otherrelay,
                            'relay' => $connectionsArray[$i]->relay,
                            'otherlong' => $connectionsArray[$i]->otherlong,
                            'otherlat' => $connectionsArray[$i]->otherlat,
                            'long' => $conn->otherlong,
                            'lat' => $conn->otherlat,
                        ]);
                        unset($connectionsArray[$i]);
                        $connectionsArray = array_values($connectionsArray);
                    } else if ($conn->otherguid == $connectionsArray[$i]->otherguid) {
                        unset($connectionsArray[$i]);
                        $connectionsArray = array_values($connectionsArray);
                    }
                }

            }
        };
        $i = 1;
        // generate the polylines.
        foreach($polylines as $polyline) {
            $i++;
            ?>
            var polyline = new google.maps.Polyline({
                path: [ new google.maps.LatLng(<?=$polyline->long?>,<?=$polyline->lat?>),
                        new google.maps.LatLng(<?=$polyline->otherlong?>,<?=$polyline->otherlat?>)],
                icons: [{
                    icon: {
                        path: google.maps.SymbolPath.FORWARD_CLOSED_ARROW,
                    },
                    offset: '100%'
                }],
                strokeColor: '#FF0000',
                title: <?=$i?>,
                strokeOpacity: 1.0,
                strokeWeight: 2,
                map: mapmap,
                geodesic: true });
            <?
        }

        foreach($markers as $marker) {
            ?>
            var marker = new google.maps.Marker({
                position: new google.maps.LatLng(<?=$marker->long?>,<?=$marker->lat?>),
                label: '<?=$marker->hop?>',
                map: mapmap
            });
            <?
        }
    ?>

  };
  </script>
<script async defer src="https://maps.googleapis.com/maps/api/js?key=AIzaSyBMLpjyj_QSi1h84r1i8SZqsM3Jzfre5aM&callback=initMap" type="text/javascript"></script>

<?
        }
  ?>
  


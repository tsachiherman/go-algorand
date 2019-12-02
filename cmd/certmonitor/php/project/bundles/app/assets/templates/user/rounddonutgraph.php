<?php $this->layout('app:layout');?>

<div class="container" style="width:81%">
    <div class="alert alert-success" style="text-align:center;font-size:30px" role="alert">
    Round <?=$roundNumber?> Authenticator Graph</div>
</div>

<style>
.donutgraphdivclass {
  position: absolute;
  width: 100%;
  height: 85%;
}
.donutcontainer {
    width:90%;
    text-align:center;
}
.tt {

}
</style>
<div class="donutcontainer" id="graph_container">
<?php
    // create a div for every relay we have.
    $level_count = count($auths);
    for($x = 0; $x < $level_count; $x++) {
        echo "<div class='donutgraphdivclass' id='donutgraph" . $x . "'></div>\r\n";
    }
?>
</div>

<script type="text/javascript">
    //var color1 = '#dff0fe';
    //var color2 = '#98ccfd';
    var color1 = '#fb9a99';
    var color2 = '#b2df8a';
    

    google.charts.load('current', {'packages':['corechart']});
    google.charts.setOnLoadCallback(drawCharts);

    /*function selectHandler(chart, data) {
        var selectedItem = chart.getSelection()[0];
        if (selectedItem) {
            alert('The user selected row ' + selectedItem.row);
        }
    };*/

    function drawCharts() {
        var centerHoleRadii = 20;
        <?
            $levelIdx = -1;
            foreach($auths as $auth) {
            
                $levelIdx++;
        ?>
        var chartDiv = document.getElementById('donutgraph<?=$levelIdx?>');
        var chartDimensions = chartDiv.getClientRects();
        for (var i = 0; i != chartDimensions.length; i++) {
            chartDimensions = chartDimensions[i];
            break;
        }
        var centerX = chartDimensions.left *0+ (chartDimensions.width / 2);
        var centerY = chartDimensions.top *0+ (chartDimensions.height / 2);
        var slice_width = (chartDimensions.height - centerHoleRadii*2) / (2*(<?=$level_count?>));

        var width = 2*(centerHoleRadii + slice_width*<?=$levelIdx?>);
        var radius = width/2;
        var data = new google.visualization.DataTable();
        data.addColumn('string', 'Auth'); 
        data.addColumn('number', 'Width');
        data.addColumn({'type': 'string', 'role': 'tooltip', 'p': {'html': true}})

        <?
            $colors = "";
            foreach($relays as $relay=>$hasrelay) {
                if (array_key_exists($relay . $auth, $relayauthmap)) {
                    $colors = $colors . "color2,";                    
                } else {
                    $colors = $colors . "color1,";
                };
                echo "data.addRows([['". $auth . "', 1, '<div class=\"tt\">Relay Name: " . $relay . "</div>']]);\r\n";
            }
            echo "var local_colors = [" .  $colors . " '#000000'];\r\n";
        ?>

        var options = {
            legend: {
                position: 'none',
            },
            pieHole: (radius-slice_width-2)/radius,
            pieSliceText: 'none',
            //pieSliceBorderColor: 'none',
            backgroundColor: { fill:'transparent' },
            'backgroundColor': 'transparent',
            colors: local_colors,
            enableInteractivity: true,
            chartArea:{
                left: centerX - radius,
                top: centerY - radius,
                width: width,
                height: width,
            },
            tooltip: {
                text: 'value',
                isHtml: true
            }
        };
        
        var chart = new google.visualization.PieChart(chartDiv);
        //google.visualization.events.addListener(chart, 'select', function () { selectHandler(chart, data); } );
        chart.draw(data, options);
        <?
            };
        ?>
    }
</script>

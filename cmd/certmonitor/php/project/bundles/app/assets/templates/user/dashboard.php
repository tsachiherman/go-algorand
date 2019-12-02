<?php $this->layout('app:layout');?>

<div class="container" style="width:81%">
    <div class="alert alert-success" style="text-align:center;font-size:30px" role="alert">Recent Rounds</div>
</div>

<div style="width:100%; text-align:center">
<div id="curve_chart" style="width: 95%; height: 400px; display:inline-block;"></div>
</div>

<div>
<table name="id" class="blueTable">
<thead><tr>
<th>Round Number</th>
<th>Connected Relay Count</th>
<th>Round Authenticators Count</th>
<th>Authenticator Distribution</th>
</tr></thead>
    <?php foreach($rounds as $round): ?>
    <tr>
    <td>
    <?=$_($round->round)?>
    </td>
    <td>
    <form style="display:inline"  id="frm<?=$round->round?>" method="GET" action="<?=$this->httpPath(
                'app.action',
                array('processor' => 'roundrelay', 'action' => 'default')
            )?>">
            <a onclick="frm<?=$round->round?>.submit()" style="cursor: pointer">
            <?=$_($round->relay_count)?>
            </a>
            <input type="text" style="display:none" name="round" value="<?=$round->round?>">
    </form>
    </td>
    <td>
    <span>
    <form style="display:inline"  id="frm_auth_<?=$round->round?>" method="GET" action="<?=$this->httpPath(
                'app.action',
                array('processor' => 'roundauthenticator', 'action' => 'default')
            )?>">
            <a onclick="frm_auth_<?=$round->round?>.submit()" style="cursor: pointer">
            <?=$_($round->auth_count)?>
            </a>
            <input type="text" style="display:none" name="round" value="<?=$round->round?>">
    </form>
    </span>
    <span style='width:20px;float: right;'>
    <form style="display:inline" id="frm_graph_<?=$round->round?>" target="_blank" method="GET" action="<?=$this->httpPath(
                'app.action',
                array('processor' => 'rounddonutgraph', 'action' => 'default')
            )?>">
            <a onclick="frm_graph_<?=$round->round?>.submit()" style="cursor: pointer">
            <img src='/bundles/app/icons8-blue-ui-40.png' width=16 height=16>
            </a>
            <input type="text" style="display:none" name="round" value="<?=$round->round?>">
    </form>
    </span>
    </td>
    <td>
    <?=$_(number_format($round->avarage_dist*100,2))?>%
    </td>
    </tr>
    <?php endforeach; ?>
    </table>
</div>


<script type="text/javascript">
      google.charts.load('current', {'packages':['corechart']});
      google.charts.setOnLoadCallback(drawChart);

      function drawChart() {
        var dataAr = [
          ['Round', 'Authenticators', 'Relays' ],
        <?php
            $round_count = 0;
            foreach($rounds as $round) {
                $round_count++;
            }
            $i = 0;
            foreach($rounds as $round) {
                echo "['" . $round->round . "', " . $round->auth_count .", " . $round->relay_count . "]";
                $i=$i+1;
                if ($i != $round_count) {
                    echo ",";
                }
                
            }
        ?>
        ];

        var data = google.visualization.arrayToDataTable(dataAr);

        var options = {
          title : 'Authenticators and Relay over rounds',
          vAxis: {title: 'Authenticators'},
          hAxis: {
              title: 'Round'
          },
          seriesType: 'bars',
          series: {1: {type: 'line'}},
          animation:{
            duration: 1000,
            easing: 'out',
            startup: true,
          }
        };

        var chart = new google.visualization.ComboChart(document.getElementById('curve_chart'));

        chart.draw(data, options);
      }
    </script>
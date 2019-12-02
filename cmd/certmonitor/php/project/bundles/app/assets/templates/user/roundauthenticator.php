<?php $this->layout('app:layout');?>

<div class="container" style="width:81%">
    <div class="alert alert-success" style="text-align:center;font-size:30px" role="alert">Authenticators for round <?=$roundNumber?></div>
</div>


<div style="width:100%; text-align:center">
<div id="curve_chart" style="width: 95%; height: 400px; display:inline-block;"></div>
</div>

<div>
<table name="id" class="blueTable">
<thead><tr>
<th>#</td>
<th>Authenticator</th>
<th>Authenticators Popularity</th>
<th>Authenticator Host GUID</th>
<th>Authenticator Host</th>
</tr></thead>
    <?php $i = 1; foreach($rounds as $round): ?>
    <tr>
    <td>
    <?=$i?>
    </td>
    <td>
    <span>
    <form style="display:inline" id="frm_auth_<?=$i?>" method="GET" action="<?=$this->httpPath(
                'app.action',
                array('processor' => 'authenticator', 'action' => 'default')
            )?>">
            <a onclick="frm_auth_<?=$i?>.submit()" style="cursor: pointer" class='authenticatorFont'>
            <?=$_($round->auth)?>
            </a>
            <input type="text" style="display:none" name="auth" value="<?=$round->auth?>">
    </form>
    </span>
    <span style='width:40px;float: right;'>
    <? if ($round->sourcehost != "") { ?>
    <form style="display:inline" id="frm_graph_<?=$i?>" target="_blank" method="GET" action="<?=$this->httpPath(
                'app.action',
                array('processor' => 'roundauthenticatorgraph', 'action' => 'default')
            )?>">
            <a onclick="frm_graph_<?=$i?>.submit()" style="cursor: pointer">
            <img src='/bundles/app/hierarchical-structure.png' width=16 height=16>
            </a>
            <input type="text" style="display:none" name="auth" value="<?=$round->auth?>">
            <input type="text" style="display:none" name="round" value="<?=$round->round?>">
            <input type="text" style="display:none" name="sourcehost" value="<?=$round->sourcehost?>">
            <input type="text" style="display:none" name="graphstyle" value="0">
    </form>
    <form style="display:inline" id="frm_graph2_<?=$i?>" target="_blank" method="GET" action="<?=$this->httpPath(
                'app.action',
                array('processor' => 'roundauthenticatorgraph', 'action' => 'default')
            )?>">
            <a onclick="frm_graph2_<?=$i?>.submit()" style="cursor: pointer">
            <img src='/bundles/app/snip.png' width=16 height=16>
            </a>
            <input type="text" style="display:none" name="auth" value="<?=$round->auth?>">
            <input type="text" style="display:none" name="round" value="<?=$round->round?>">
            <input type="text" style="display:none" name="sourcehost" value="<?=$round->sourcehost?>">
            <input type="text" style="display:none" name="graphstyle" value="1">
    </form>
    <? } ?>
    </span>

    </td>
    <td>
    <?
        if ($round->dist >= 0.75) {
            echo "<span style='color:green'>";
        } else if ($round->dist < 0.1) {
            echo "<span style='color:red'>";
        } else {
            echo "<span>";
        }
    ?>
    <?=$_(number_format($round->dist*100,2))?>%
    </span>&nbsp(
    <?=$_($round->auth_count)?>&nbspout&nbspof&nbsp<?=$_($round->auth_count/$round->dist)?>)
    </td>
    <td>
    <?
        $explode_source_host = explode(":", $round->sourcehost);
        if (count($explode_source_host) > 0) {
            echo $explode_source_host[0];
        }
    ?>
    </td>
    <td>
    <?
        if (count($explode_source_host) > 1) {
            echo $explode_source_host[1];
        }
    ?>
    </td>
    </tr>
    <?php $i=$i+1; endforeach; ?>
    </table>
</div>

<script type="text/javascript">
      google.charts.load('current', {'packages':['corechart']});
      google.charts.setOnLoadCallback(drawChart);

      function drawChart() {
        var dataAr = [
          ['Popularity Bucket', 'Authenticator Count'],
        <?php
            $first = 1;
            foreach($rounds as $round) {
                if ($first != 1) {
                    echo ",";
                }
                $first = 0;
                echo "['Authenticator " . $round->auth . "', ";
                echo number_format($round->dist*100-1,2) . "] ";
            }
        ?>
        ]

        var data = google.visualization.arrayToDataTable(dataAr);

        var options = {
          title: 'Authenticator Distribution Histogram',
          titlePosition: 'out',
          legend: { position: 'bottom' },
          animation:{
            duration: 1000,
            easing: 'out',
            startup: true,
          },
          hAxis: {
            ticks: [5, 10, 15, 20, 25, 30, 35, 40, 45, 50, 55, 60, 65, 70, 75, 80, 85, 90, 95, 100],
            viewWindow: {
                min: 0,
                max: 100
            }
          },
          bar: { gap: 0 },
          histogram: {
              bucketSize: 5,
              maxNumBuckets: 20,
              hideBucketItems: true,
          }
        };

        var chart = new google.visualization.Histogram(document.getElementById('curve_chart'));

        chart.draw(data, options);
      }
    </script>

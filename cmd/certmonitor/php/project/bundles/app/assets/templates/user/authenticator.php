<?php $this->layout('app:layout');?>

<div class="container" style="width:81%">
    <div class="alert alert-success" style="text-align:center;font-size:30px" role="alert">Authenticator <?=$auth?> Statistics</div>
</div>


<div style="width:100%; text-align:center">
<div id="curve_chart" style="width: 95%; height: 400px; display:inline-block;"></div>
</div>

<div>
<table name="id" class="blueTable">
<thead><tr>
<th>#</td>
<th>Round</th>
<th>Popularity</th>
<th>Authenticator Host GUID</th>
<th>Authenticator Host</th>
</tr></thead>
    <?php $i = 1; foreach($authsDist as $authDist): ?>
    <tr>
    <td>
    <?=$i?>
    </td>
    <td>
    <form id="frm_auth_<?=$authDist->round?>" method="GET" action="<?=$this->httpPath(
                'app.action',
                array('processor' => 'roundauthenticator', 'action' => 'default')
            )?>">
            <a onclick="frm_auth_<?=$authDist->round?>.submit()" style="cursor: pointer">
            <?=$_($authDist->round)?>
            </a>
            <input type="text" style="display:none" name="round" value="<?=$authDist->round?>">
    </form>
    </td>
    <td>
    <?
        if ($authDist->dist >= 0.75) {
            echo "<span style='color:green'>";
        } else if ($authDist->dist < 0.1) {
            echo "<span style='color:red'>";
        } else {
            echo "<span>";
        }
    ?>
    <?=$_(number_format($authDist->dist*100,2))?>%
    </span>&nbsp(
    <?=$_($authDist->auth_count)?>&nbspout&nbspof&nbsp<?=$_($authDist->auth_count/$authDist->dist)?>)
    </td>
    <td>
    <?
        $explode_source_host = explode(":", $authDist->sourcehost);
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
          ['Round', 'Round Popularity', 'Overall Avarage Popularity'],
        <?php
            $i = 0;
            $out = "";
            foreach($authsDist as $authDist) {
                $itemOut = "";
                $itemOut = $itemOut . "['" . $authDist->round . "', ";
                $itemOut = $itemOut . number_format($authDist->dist*100,2) . ", ";
                $itemOut = $itemOut . number_format($averageDist*100,2) . "]";
                $i++;
                if ($i != 0) {
                  $itemOut = $itemOut . ",";
                }
                $out = $itemOut . $out;
            }
            echo $out
        ?>
        ];

        var data = google.visualization.arrayToDataTable(dataAr);

        var options = {
          title : 'Authenticator Popularity across rounds',
          vAxis: {title: 'Round Popularity'},
          hAxis: {
              title: 'Round'
          },
          seriesType: 'line',
          series: {1: {type: 'line'}},
          pointsVisible: true,
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

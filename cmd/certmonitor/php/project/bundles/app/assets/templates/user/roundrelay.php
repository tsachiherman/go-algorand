<?php $this->layout('app:layout');?>

<div class="container" style="width:81%">
    <div class="alert alert-success" style="text-align:center;font-size:30px" role="alert">Relay information available for round <?=$roundNumber?></div>
</div>

<div>
<table name="id" class="blueTable">
<thead><tr>
<th>#</td>
<th>Relay</th>
<th>Authenticators Count</th>
</tr></thead>
    <?php $i = 1; foreach($rounds as $round): ?>
    <tr>
    <td>
    <?=$i?>
    </td>
    <td >
    <?=$_($round->relay)?>
    <?
        $exp_relay = explode(":", $round->relay);
        if (count($exp_relay) > 1) {
            echo "<a href='https://api.hackertarget.com/geoip/?q=" . $exp_relay[0]. "' target=_blank><img src='/bundles/app/location.png' height='16px'/></a>";
        }
    ?>
    </td>
    <td>
    <?=$_($round->auth_count)?>
    </td>
    </tr>
    <?php $i=$i+1; endforeach; ?>
    </table>
</div>
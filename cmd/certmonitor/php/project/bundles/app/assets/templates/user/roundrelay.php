<?php $this->layout('app:layout');?>

<div class="container" style="width:81%">
    <div class="alert alert-success" style="text-align:center;font-size:30px" role="alert">Relay information available for round <?=$roundNumber?></div>
</div>

<div>
<table name="id" class="blueTable">
<thead><tr>
<th>#</td>
<th>Relay</th>
<th>Relay Location</th>
<th>Authenticators Count</th>
</tr></thead>
    <?php $i = 1; foreach($rounds as $round): ?>
    <tr>
    <td>
    <?=$i?>
    </td>
    <td >
    <?=$_($round->relay)?>
    </td>
    <td>
    <?php
        $s = "";
        if ($round->country != "") {
            $s = $round->country;
        }
        if ($round->state != "") {
            if ($s != "") {
                $s = $s . ", ";
            }
            $s = $s . $round->state;
        }
        if ($round->city != "") {
            if ($s != "") {
                $s = $s . ", ";
            }
            $s = $s . $round->city;
        }
        echo "<span>" . $s . "</span>";
        if ($round->lat != "") {
            echo "<span style='width:20px;float: right;'><a href='https://maps.google.com/?q=" . $round->long . "," . $round->lat . "' target=_blank><img src='/bundles/app/location.png' height='16px'/></a></span>";
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
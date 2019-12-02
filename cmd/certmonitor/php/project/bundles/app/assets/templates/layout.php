<?php $this->layout('app:html');?>
<nav class="navbar navbar-default">
	<div class="container-fluid">
		<div class="navbar-header">
			<a class="navbar-brand" href="/">
			<img src="/bundles/app/Certificate128.png" style='position:absolute;transform: translateY(-50%);-ms-transform: translateY(-50%);top: 50%;width:32px; height:32px'/>
			</a>
			<a class="navbar-brand" href="/" style='margin-left:8px'>
			<span>Algorand Relay Certificate Explorer</span>
			</a>
		</div>

		<div class="nav navbar-nav navbar-right">
			<?php if($user): ?>
				<p class="navbar-text">
					<?=$user->email?>
					<a class="navbar-link" href="<?=$this->httpPath(
						'app.action',
						array('processor' => 'auth', 'action' => 'logout')
					)?>"> <span class="glyphicon glyphicon-log-out"></span></a>
				</p>
			<?php else: ?>
				<li><a href="<?=$this->httpPath(
						'app.processor',
						array('processor' => 'auth')
					)?>">Login</a></li>
			<?php endif;?>
		</div>
	</div>
</nav>

<?php $this->childContent(); ?>
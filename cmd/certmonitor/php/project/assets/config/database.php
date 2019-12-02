<?php
$root = realpath(__DIR__.'/../../');

return array(
    'default' => array(
        'driver' => 'pdo',
        'connection' => 'pgsql:host=localhost;dbname=relays',
        'user'       => 'tsachi',
        'password'   => 'gogators'
    )
);
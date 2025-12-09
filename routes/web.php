<?php

use BareMetalPHP\Routing\Router;
use BareMetalPHP\Http\Response;
use BareMetalPHP\View\View;

return function (Router $router) {
    $router->get('/', function() {
        return View::make('welcome');
    });

    $router->get('/test', function() {
        return new Response("We did it!!");
    });

    $router->get('/stream/logs', function () {

        return new Response(
            200,
            ['Content-Type' => 'text/plain'],
            "Streaming logs demo\nLine 1\nLine 2\n"
        );

    });
};


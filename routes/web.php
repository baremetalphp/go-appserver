<?php

use BareMetalPHP\Routing\Router;
use BareMetalPHP\Http\Response;

return function (Router $router) {
    $router->get('/', function() {
        return "Hello from BareMetalPHP real router version B!\n!";
    });

    $router->get('/test', function() {
        return new Response("We did it!!");
    });
};


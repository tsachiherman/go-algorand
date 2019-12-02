<?php

namespace Project\App\HTTPProcessors;

use PHPixie\HTTP\Request;
use Project\App\HTTPProcessors\Processor\UserProtected;

/**
 * User roundrelay
 */
class Authenticator extends UserProtected
{
    /**
     * @param Request $request
     * @return mixed
     */
    public function defaultAction(Request $request)
    {
        // find the average.
        $auth = $request->query()->get('auth');

        $authsDist = $this->authdist()->query()->where('auth', '=', $auth)->orderDescendingBy("round")->limit(256)->find();

        $authsDistAvg = $this->authdistavg()->query()->find();
        $avg = 0;
        foreach($authsDistAvg as $authDistAvg) {
            $avg = $authDistAvg->avgdist;
        }

        return $this->components->template()->get('app:user/authenticator', array(
            'user' => $this->user,
            'authsDist' => $authsDist,
            'auth' => $auth,
            'averageDist' => $avg
        ));
    }

    /**
     * @return UserRepository
     */
    protected function authdist()
    {
        return $this->components->orm()->repository('authdist');
    }

    /**
     * @return UserRepository
     */
    protected function authdistavg()
    {
        return $this->components->orm()->repository('authdistavg');
    }

}
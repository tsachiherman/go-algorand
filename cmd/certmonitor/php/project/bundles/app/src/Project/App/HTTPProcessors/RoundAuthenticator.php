<?php

namespace Project\App\HTTPProcessors;

use PHPixie\HTTP\Request;
use Project\App\HTTPProcessors\Processor\UserProtected;

/**
 * User roundrelay
 */
class RoundAuthenticator extends UserProtected
{
    /**
     * @param Request $request
     * @return mixed
     */
    public function defaultAction(Request $request)
    {
        $roundNumber = $request->query()->get('round');

        $auths = $this->authdist()->query()->where('round', '=', intval($roundNumber))->orderAscendingBy("dist")->find();

        return $this->components->template()->get('app:user/roundauthenticator', array(
            'user' => $this->user,
            'rounds' => $auths,
            'roundNumber' => $roundNumber
        ));
    }

    /**
     * @return UserRepository
     */
    protected function authdist()
    {
        return $this->components->orm()->repository('authdistex');
    }
}
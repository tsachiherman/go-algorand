<?php

namespace Project\App\HTTPProcessors;

use PHPixie\HTTP\Request;
use Project\App\HTTPProcessors\Processor\UserProtected;

/**
 * User roundrelay
 */
class RoundRelay extends UserProtected
{
    /**
     * @param Request $request
     * @return mixed
     */
    public function defaultAction(Request $request)
    {
        $roundNumber = $request->query()->get('round');

        $auths = $this->roundrelay()->query()->where('round', '=', intval($roundNumber))->orderAscendingBy("relay")->find();

        return $this->components->template()->get('app:user/roundrelay', array(
            'user' => $this->user,
            'rounds' => $auths,
            'roundNumber' => $roundNumber
        ));
    }

    /**
     * @return UserRepository
     */
    protected function roundrelay()
    {
        return $this->components->orm()->repository('roundrelayex');
    }
}
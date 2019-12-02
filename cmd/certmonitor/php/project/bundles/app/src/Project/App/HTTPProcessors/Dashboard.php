<?php

namespace Project\App\HTTPProcessors;

use PHPixie\HTTP\Request;
use Project\App\HTTPProcessors\Processor\UserProtected;

/**
 * User dashboard
 */
class Dashboard extends UserProtected
{
    /**
     * @param Request $request
     * @return mixed
     */
    public function defaultAction(Request $request)
    {
        $roundsQuery = $this->rounds()->query();
        $roundsQuery->orderDescendingBy("round");
        $roundsQuery->limit(20);
        $rounds = $roundsQuery->find();

        return $this->components->template()->get('app:user/dashboard', array(
            'user' => $this->user,
            'rounds' => $rounds
        ));
    }

    /**
     * @return UserRepository
     */
    protected function rounds()
    {
        return $this->components->orm()->repository('rounddist');
    }
}
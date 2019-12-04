<?php

namespace Project\App\HTTPProcessors;

use PHPixie\HTTP\Request;
use Project\App\HTTPProcessors\Processor\UserProtected;

/**
 * User roundrelay
 */
class RoundDonutGraph extends UserProtected
{
    /**
     * @param Request $request
     * @return mixed
     */
    public function defaultAction(Request $request)
    {
        $roundNumber = $request->query()->get('round');
        $graphStyle = $request->query()->get('graphstyle');
        

        // retrieve the connections map for the above timestamp.
        $relayauthenticators = $this->auth()->query()-> 
            where('round', '=', $roundNumber)->
            orderby('relay', 'asc')->
            find();

        $relays = array();
        $auths = array();
        $relayauthmap = array();
        foreach($relayauthenticators as $relayauth) {
            $relays[$relayauth->relay] = true;
            //$auths[$relayauth->auth] = true;
            $relayauthmap[$relayauth->relay . $relayauth->auth] = $relayauth;
        }

        $authdists = $this->authdist()->query()->
            where('round', '=', $roundNumber)->
            orderby('dist', 'desc')->
            find();

        foreach($authdists as $authdist) {
            array_push($auths, $authdist->auth);
        };
        

        return $this->components->template()->get('app:user/rounddonutgraph', array(
            'user' => $this->user,
            'roundNumber' => $roundNumber,
            'relays' => $relays,
            'auths' => $auths,
            'relayauthmap' => $relayauthmap,
            'graphStyle' => $graphStyle
        ));
    }

    /**
     * @return UserRepository
     */
    protected function auth()
    {
        return $this->components->orm()->repository('authenticator');
    }

    /**
     * @return UserRepository
     */
    protected function authdist()
    {
        return $this->components->orm()->repository('authdist');
    }
}
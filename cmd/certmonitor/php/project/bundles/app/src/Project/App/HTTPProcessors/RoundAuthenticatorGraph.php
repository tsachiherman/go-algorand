<?php

namespace Project\App\HTTPProcessors;

use PHPixie\HTTP\Request;
use Project\App\HTTPProcessors\Processor\UserProtected;

/**
 * User roundrelay
 */
class RoundAuthenticatorGraph extends UserProtected
{
    /**
     * @param Request $request
     * @return mixed
     */
    public function defaultAction(Request $request)
    {
        $roundNumber = $request->query()->get('round');
        $auth = $request->query()->get('auth');
        $sourcehost = $request->query()->get('sourcehost');
        $graphstyle = $request->query()->get('graphstyle');

        if ($sourcehost == '') {
            // we don't have any information regarding the source of the vote.
            $votes = $this->vote()->query()->
                where('round', '=', intval($roundNumber))->
                andWhere('step', '=', 2)->
                limit(1)->
                find();
            
            foreach($votes as $vote) {
                $firstVote = $vote;
                break;
            }
            $firstVoteTimestamp = $firstVote->timestamp;

        } else {
            // we know from which host the transaction came from.
            $votes = $this->vote()->query()->
                where('round', '=', intval($roundNumber))->
                andWhere('sender', '=', $auth)->
                andWhere('step', '=', 2)->
                limit(1)->
                find();
            
            foreach($votes as $vote) {
                $firstVote = $vote;
                break;
            }
            $firstVoteTimestamp = $firstVote->timestamp;
        }

        $startTimestamp = (new \DateTime($firstVoteTimestamp))->sub(new \DateInterval('PT1H'));
        $firstVoteTimestamp = new \DateTime($firstVoteTimestamp);

        $firstVoteTimestamp = $firstVoteTimestamp->format("Y-m-d H:i:s") . '-05';
        $startTimestamp = $startTimestamp->format("Y-m-d H:i:s") . '-05';

        // retrieve the connections map for the above timestamp.
        $connections = $this->relayconnection()->query()-> 
            where('quanttime', 'between', $startTimestamp, $firstVoteTimestamp)->
            find();

        if ($sourcehost != "") {
            $explode_source_host = explode(":", $sourcehost);
            $source_guid = '';
            if (count($explode_source_host) > 0) {
                $source_guid = $explode_source_host[0];
                $voterConnections = $this->connection()->query()-> 
                    where('quanttime', 'between', $startTimestamp, $firstVoteTimestamp)->
                    andWhere('guid', '=', $source_guid)->
                    find();
            } else {
                $source_guid = "unknown";
                $voterConnections = $this->connection()->query()-> 
                    where('quanttime', 'between', $startTimestamp, $firstVoteTimestamp)->
                    andWhere('name', '=', $sourcehost)->
                    find();
            }
            
            // count voter connections
            $vcCount = 0;
            foreach($voterConnections as $vc) {
                $vcCount++;
            }
            if ($vcCount == 0) {
                // generate a stub:
                $voterConnections = [
                    (object) [
                        'name' => $sourcehost,
                        'guid' => $source_guid,
                        'otherguid' => '',
                        'othername' => ''
                    ]
                    ];
            }
        } else {
            $voterConnections = [];
        }

        $seenAuthRelays = $this->auth()->query()->
            where('round', '=', $roundNumber)->
            andWhere('auth', '=', $auth)->
            find();

        return $this->components->template()->get('app:user/roundauthenticatorgraph', array(
            'user' => $this->user,
            'auth' => $auth,
            'roundNumber' => $roundNumber,
            'sourcehost' => $sourcehost,
            'firstVoteTimestamp' => $firstVoteTimestamp,
            'startTimestamp' => $startTimestamp,
            'connections' => $connections,
            'voterConnections' => $voterConnections,
            'seenAuthRelays' => $seenAuthRelays,
            'graphstyle' => $graphstyle,
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
    protected function vote()
    {
        return $this->components->orm()->repository('vote');
    }

    /**
     * @return UserRepository
     */
    protected function connection()
    {
        return $this->components->orm()->repository('connection');
    }

     /**
     * @return UserRepository
     */
    protected function relayconnection()
    {
        return $this->components->orm()->repository('relayconnection');
    }
}
<?php

namespace Project\App;

/**
 * Handles processing of the HTTP requests
 */
class HTTPProcessor extends \PHPixie\DefaultBundle\Processor\HTTP\Builder
{
    /**
     * @var Builder
     */
    protected $builder;

    /**
     * Constructor
     * @param Builder $builder
     */
    public function __construct($builder)
    {
        $this->builder = $builder;
    }

    /**
     * Build 'greet' processor
     * @return HTTPProcessors\Auth
     */
    protected function buildAuthProcessor()
    {
        return new HTTPProcessors\Auth(
            $this->builder
        );
    }

    /**
     * Build 'dashboard' processor
     * @return HTTPProcessors\Greet
     */
    protected function buildDashboardProcessor()
    {
        return new HTTPProcessors\Dashboard($this->builder);
    }

    /**
     * Build 'roundrelay' processor
     * @return HTTPProcessors\Greet
     */
    protected function buildRoundRelayProcessor()
    {
        return new HTTPProcessors\RoundRelay($this->builder);
    }

    /**
     * Build 'rounddonutgraph' processor
     * @return HTTPProcessors\Greet
     */
    protected function buildRoundDonutGraphProcessor()
    {
        return new HTTPProcessors\RoundDonutGraph($this->builder);
    }

    /**
     * Build 'roundauthenticator' processor
     * @return HTTPProcessors\Greet
     */
    protected function buildRoundAuthenticatorProcessor()
    {
        return new HTTPProcessors\RoundAuthenticator($this->builder);
    }

    /**
     * Build 'roundauthenticatorgraph' processor
     * @return HTTPProcessors\Greet
     */
    protected function buildRoundAuthenticatorGraphProcessor()
    {
        return new HTTPProcessors\RoundAuthenticatorGraph($this->builder);
    }

    /**
     * Build 'authenticator' processor
     * @return HTTPProcessors\Greet
     */
    protected function buildAuthenticatorProcessor()
    {
        return new HTTPProcessors\Authenticator($this->builder);
    }

    /**
     * Build 'frontpage' processor
     * @return HTTPProcessors\Greet
     */
    protected function buildFrontpageProcessor()
    {
        return new HTTPProcessors\Frontpage($this->builder);
    }

    /**
     * Build 'admin' processor group
     * @return HTTPProcessors\Admin\
     */
    protected function buildAdminProcessor()
    {
        return new HTTPProcessors\AdminProcessors($this->builder);
    }
}
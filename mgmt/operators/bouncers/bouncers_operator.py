import random
import logging
from kubernetes import client, config
from obj.bouncer import Bouncer
from obj.divider import Divider
from obj.endpoint import Endpoint
from common.constants import *
from common.common import *
from obj.net import Net
from store.operator_store import OprStore

logger = logging.getLogger()

class BouncerOperator(object):
	_instance = None

	def __new__(cls, **kwargs):
		if cls._instance is None:
			cls._instance = super(BouncerOperator, cls).__new__(cls)
			cls._init(cls, **kwargs)
		return cls._instance

	def _init(self, **kwargs):
		logger.info(kwargs)
		self.store = OprStore()
		config.load_incluster_config()
		self.obj_api = client.CustomObjectsApi()

	def on_startup(self, logger, **kwargs):
		logger.info("bouncer on_startup")
		def list_bouncers_obj_fn(name, spec, plurals):
			logger.info("Bootstrapped Bouncer {}".format(name))
			b = Bouncer(name, self.obj_api, self.store, spec)
			self.store.update_bouncer(b)

		kube_list_obj(self.obj_api, RESOURCES.droplets, list_bouncers_obj_fn)

	def on_bouncer_provisioned(self, body, spec, **kwargs):
		name = kwargs['name']
		logger.info("on_bouncer_provisioned {}".format(spec))
		b = Bouncer(name, self.obj_api, self.store, spec)
		self.store.update_bouncer(b)
		logger.info("on_bouncer_provisioned Dump bouncers start ------------")
		self.store._dump_bouncers()
		logger.info("on_bouncer_provisioned Dump bouncers end ------------")

	def on_divider_placed(self, body, spec, **kwargs):
		name = kwargs['name']
		logger.info("on_divider_placed {}".format(spec))
		# divider.bouncers = net.get_all_bouncers (bouncers_opr)
		div = Divider(name, self.obj_api, None, spec)
		bouncers = self.store.get_bouncers_of_vpc(div.vpc)
		div.update_bouncers(set(bouncers))
		for b in bouncers:
			b.update_dividers(set([div]))
		div.set_status(OBJ_STATUS.divider_status_provisioned)
		div.update_obj()

	def on_endpoints_allocated(self, body, spec, **kwargs):
		name = kwargs['name']
		logger.info("on_endpoints_allocated {}".format(spec))
		ep = Endpoint(name, self.obj_api, None, spec)
		bouncers = self.store.get_bouncers_of_net(ep.net)
		logger.info("on_endpoints_allocated Dump bouncers {}".format(bouncers))
		eps = set([ep])
		for b in bouncers:
			b.update_eps(eps)
		ep.update_bouncers(bouncers)
		ep.update_md()
		ep.set_status(OBJ_STATUS.ep_status_provisioned)
		ep.update_obj()